package httpinterface

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
)

type httpMetricKey struct {
	method string
	path   string
	status int
}

type targetMetricKey struct {
	operation string
	outcome   string
}

type durationHistogram struct {
	buckets [11]atomic.Uint64
	count   atomic.Uint64
	nanos   atomic.Int64
}

var durationBuckets = [...]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type Metrics struct {
	httpRequests sync.Map
	durations    sync.Map
	target       sync.Map
	changes      atomic.Uint64
	proposals    atomic.Uint64
	evaluations  [2]atomic.Uint64
	releases     [2]atomic.Uint64
	rollbacks    atomic.Uint64
}

func NewMetrics() *Metrics { return &Metrics{} }

func (metrics *Metrics) ChangeAccepted()  { metrics.changes.Add(1) }
func (metrics *Metrics) ProposalCreated() { metrics.proposals.Add(1) }
func (metrics *Metrics) EvaluationCompleted(passed bool) {
	index := 0
	if passed {
		index = 1
	}
	metrics.evaluations[index].Add(1)
}
func (metrics *Metrics) ReleaseCompleted(outcome string) {
	index := 0
	if outcome == "success" {
		index = 1
	}
	metrics.releases[index].Add(1)
}
func (metrics *Metrics) RollbackCreated() { metrics.rollbacks.Add(1) }
func (metrics *Metrics) TargetRequest(operation, outcome string) {
	value, _ := metrics.target.LoadOrStore(targetMetricKey{operation, outcome}, &atomic.Uint64{})
	value.(*atomic.Uint64).Add(1)
}

func (metrics *Metrics) observeHTTP(method, pattern string, status int, duration time.Duration) {
	if pattern == "" {
		pattern = "unmatched"
	}
	value, _ := metrics.httpRequests.LoadOrStore(httpMetricKey{method, pattern, status}, &atomic.Uint64{})
	value.(*atomic.Uint64).Add(1)
	histogramValue, _ := metrics.durations.LoadOrStore(pattern, &durationHistogram{})
	histogram := histogramValue.(*durationHistogram)
	seconds := duration.Seconds()
	for index, upper := range durationBuckets {
		if seconds <= upper {
			histogram.buckets[index].Add(1)
		}
	}
	histogram.count.Add(1)
	histogram.nanos.Add(duration.Nanoseconds())
}

func metricEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func metricLabels(values ...string) string {
	parts := make([]string, 0, len(values)/2)
	for index := 0; index < len(values); index += 2 {
		parts = append(parts, values[index]+`="`+metricEscape(values[index+1])+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func writeMetricHeader(builder *strings.Builder, name, help, kind string) {
	fmt.Fprintf(builder, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, kind)
}

func (metrics *Metrics) render(health engine.SystemHealth) string {
	var builder strings.Builder
	writeMetricHeader(&builder, "gyrifi_http_requests_total", "Total HTTP requests handled.", "counter")
	var requests []struct {
		key   httpMetricKey
		value uint64
	}
	metrics.httpRequests.Range(func(key, value any) bool {
		requests = append(requests, struct {
			key   httpMetricKey
			value uint64
		}{key.(httpMetricKey), value.(*atomic.Uint64).Load()})
		return true
	})
	sort.Slice(requests, func(i, j int) bool {
		left, right := requests[i].key, requests[j].key
		if left.method != right.method {
			return left.method < right.method
		}
		if left.path != right.path {
			return left.path < right.path
		}
		return left.status < right.status
	})
	for _, request := range requests {
		fmt.Fprintf(&builder, "gyrifi_http_requests_total%s %d\n", metricLabels("method", request.key.method, "path_template", request.key.path, "status", strconv.Itoa(request.key.status)), request.value)
	}

	writeMetricHeader(&builder, "gyrifi_http_request_duration_seconds", "HTTP request duration in seconds.", "histogram")
	var paths []string
	metrics.durations.Range(func(key, _ any) bool {
		paths = append(paths, key.(string))
		return true
	})
	sort.Strings(paths)
	for _, path := range paths {
		value, _ := metrics.durations.Load(path)
		histogram := value.(*durationHistogram)
		for index, upper := range durationBuckets {
			fmt.Fprintf(&builder, "gyrifi_http_request_duration_seconds_bucket%s %d\n", metricLabels("path_template", path, "le", strconv.FormatFloat(upper, 'g', -1, 64)), histogram.buckets[index].Load())
		}
		count := histogram.count.Load()
		fmt.Fprintf(&builder, "gyrifi_http_request_duration_seconds_bucket%s %d\n", metricLabels("path_template", path, "le", "+Inf"), count)
		fmt.Fprintf(&builder, "gyrifi_http_request_duration_seconds_sum%s %.9f\n", metricLabels("path_template", path), float64(histogram.nanos.Load())/float64(time.Second))
		fmt.Fprintf(&builder, "gyrifi_http_request_duration_seconds_count%s %d\n", metricLabels("path_template", path), count)
	}

	writeMetricHeader(&builder, "gyrifi_changes_accepted_total", "Total Changes durably accepted.", "counter")
	fmt.Fprintf(&builder, "gyrifi_changes_accepted_total %d\n", metrics.changes.Load())
	writeMetricHeader(&builder, "gyrifi_proposals_created_total", "Total Proposals durably created.", "counter")
	fmt.Fprintf(&builder, "gyrifi_proposals_created_total %d\n", metrics.proposals.Load())
	writeMetricHeader(&builder, "gyrifi_evaluations_total", "Total persisted evaluations by result.", "counter")
	fmt.Fprintf(&builder, "gyrifi_evaluations_total%s %d\n", metricLabels("passed", "false"), metrics.evaluations[0].Load())
	fmt.Fprintf(&builder, "gyrifi_evaluations_total%s %d\n", metricLabels("passed", "true"), metrics.evaluations[1].Load())
	writeMetricHeader(&builder, "gyrifi_releases_total", "Total Release finalization outcomes.", "counter")
	fmt.Fprintf(&builder, "gyrifi_releases_total%s %d\n", metricLabels("outcome", "failure"), metrics.releases[0].Load())
	fmt.Fprintf(&builder, "gyrifi_releases_total%s %d\n", metricLabels("outcome", "success"), metrics.releases[1].Load())
	writeMetricHeader(&builder, "gyrifi_rollbacks_total", "Total rollback Proposals durably created.", "counter")
	fmt.Fprintf(&builder, "gyrifi_rollbacks_total %d\n", metrics.rollbacks.Load())

	writeMetricHeader(&builder, "gyrifi_target_requests_total", "Total target adapter operations by outcome.", "counter")
	var targets []struct {
		key   targetMetricKey
		value uint64
	}
	metrics.target.Range(func(key, value any) bool {
		targets = append(targets, struct {
			key   targetMetricKey
			value uint64
		}{key.(targetMetricKey), value.(*atomic.Uint64).Load()})
		return true
	})
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].key.operation != targets[j].key.operation {
			return targets[i].key.operation < targets[j].key.operation
		}
		return targets[i].key.outcome < targets[j].key.outcome
	})
	for _, target := range targets {
		fmt.Fprintf(&builder, "gyrifi_target_requests_total%s %d\n", metricLabels("operation", target.key.operation, "outcome", target.key.outcome), target.value)
	}

	writeMetricHeader(&builder, "gyrifi_unresolved_intents", "Release Intents requiring operator recovery.", "gauge")
	fmt.Fprintf(&builder, "gyrifi_unresolved_intents %d\n", health.UnresolvedIntents)
	writeMetricHeader(&builder, "gyrifi_object_store_bytes", "Bytes stored in durable content-addressed objects.", "gauge")
	fmt.Fprintf(&builder, "gyrifi_object_store_bytes %d\n", health.ObjectStoreBytes)
	writeMetricHeader(&builder, "gyrifi_pending_changes", "Accepted or Ready Changes not yet released.", "gauge")
	fmt.Fprintf(&builder, "gyrifi_pending_changes %d\n", health.PendingChanges)
	writeMetricHeader(&builder, "gyrifi_build_info", "Build identity for this Gyrifi process.", "gauge")
	fmt.Fprintf(&builder, "gyrifi_build_info%s 1\n", metricLabels("version", buildinfo.Version, "commit", buildinfo.Commit))
	return builder.String()
}

func (server *Server) MetricsHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = fmt.Fprint(writer, server.metrics.render(server.health.current()))
	})
	mux.HandleFunc("/", func(writer http.ResponseWriter, _ *http.Request) {
		server.writeError(writer, engine.CodeNotFound, "Endpoint was not found.", http.StatusNotFound)
	})
	return mux
}
