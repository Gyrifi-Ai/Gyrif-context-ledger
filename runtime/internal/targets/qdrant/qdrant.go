package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/ledger"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets"
)

type Adapter struct {
	baseURL    *url.URL
	collection string
	apiKey     string
	client     *http.Client
}

func New(rawURL, collection, apiKey string) (*Adapter, error) {
	parsed, err := url.Parse(strings.TrimRight(rawURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Qdrant URL")
	}
	if collection == "" {
		return nil, fmt.Errorf("Qdrant collection is required")
	}
	return &Adapter{baseURL: parsed, collection: collection, apiKey: apiKey, client: &http.Client{Timeout: 20 * time.Second}}, nil
}

func (adapter *Adapter) endpoint(path string) string {
	return adapter.baseURL.String() + "/collections/" + url.PathEscape(adapter.collection) + path
}
func (adapter *Adapter) Health(ctx context.Context) error {
	_, err := adapter.request(ctx, http.MethodGet, "", nil)
	return err
}
func (adapter *Adapter) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, adapter.endpoint(path), reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if adapter.apiKey != "" {
		request.Header.Set("api-key", adapter.apiKey)
	}
	response, err := adapter.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: Qdrant request failed: %v", targets.ErrUnavailable, err)
	}
	defer response.Body.Close()
	value, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, targets.ErrNotFound
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: Qdrant returned status %d", targets.ErrAuthentication, response.StatusCode)
	}
	if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusConflict || response.StatusCode == http.StatusUnprocessableEntity {
		return nil, fmt.Errorf("%w: Qdrant returned status %d", targets.ErrSemantic, response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: Qdrant returned status %d", targets.ErrUnavailable, response.StatusCode)
	}
	return value, nil
}

func (adapter *Adapter) Read(ctx context.Context, unit string) (targets.Value, error) {
	body, err := adapter.request(ctx, http.MethodGet, "/points/"+url.PathEscape(unit), nil)
	if errors.Is(err, targets.ErrNotFound) {
		return targets.Value{Unit: unit, Exists: false}, nil
	}
	if err != nil {
		return targets.Value{}, err
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return targets.Value{}, fmt.Errorf("decode Qdrant point: %w", err)
	}
	logicalValue, err := normalizePoint(envelope.Result)
	if err != nil {
		return targets.Value{}, fmt.Errorf("normalize Qdrant point: %w", err)
	}
	return targets.Value{Unit: unit, Value: logicalValue, Fingerprint: ledger.Fingerprint(logicalValue), Exists: true}, nil
}
func (adapter *Adapter) Fingerprint(ctx context.Context, unit string) (string, error) {
	value, err := adapter.Read(ctx, unit)
	if err != nil {
		return "", err
	}
	return value.Fingerprint, nil
}
func (adapter *Adapter) Preview(_ context.Context, changes []ledger.Change) (targets.Preview, error) {
	return targets.Preview{Fidelity: "FAST", Summary: fmt.Sprintf("Qdrant overlay contains %d logical changes", len(changes))}, nil
}
func (adapter *Adapter) Compile(ctx context.Context, changes []ledger.Change) (targets.Plan, error) {
	plan := targets.Plan{Operations: make([]targets.Operation, 0, len(changes))}
	distance, err := adapter.distance(ctx)
	if err != nil {
		return targets.Plan{}, err
	}
	for _, change := range changes {
		desiredFingerprint := change.DesiredFingerprint
		if change.Action == ledger.ChangePut {
			logicalValue, err := normalizePoint(change.Desired)
			if err != nil {
				return targets.Plan{}, fmt.Errorf("normalize desired Qdrant point %s: %w", change.Unit, err)
			}
			desiredFingerprint = ledger.Fingerprint(logicalValue)
		}
		plan.Operations = append(plan.Operations, targets.Operation{Unit: change.Unit, Action: change.Action, Desired: change.Desired, ExpectedFingerprint: change.BaseFingerprint, DesiredFingerprint: desiredFingerprint, TargetMetric: distance})
	}
	return plan, nil
}

func (adapter *Adapter) distance(ctx context.Context) (string, error) {
	body, err := adapter.request(ctx, http.MethodGet, "", nil)
	if err != nil {
		return "", fmt.Errorf("read Qdrant collection configuration: %w", err)
	}
	var envelope struct {
		Result struct {
			Config struct {
				Params struct {
					Vectors struct {
						Distance string `json:"distance"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("decode Qdrant collection configuration: %w", err)
	}
	if envelope.Result.Config.Params.Vectors.Distance == "" {
		return "", fmt.Errorf("named or missing Qdrant vector configuration is not supported")
	}
	return envelope.Result.Config.Params.Vectors.Distance, nil
}

func normalizePoint(value json.RawMessage) (json.RawMessage, error) {
	var point map[string]json.RawMessage
	if err := json.Unmarshal(value, &point); err != nil {
		return nil, err
	}
	logical := make(map[string]json.RawMessage, 3)
	for _, field := range []string{"id", "vector", "payload"} {
		if fieldValue, exists := point[field]; exists {
			logical[field] = fieldValue
		}
	}
	if len(logical) == 0 {
		return nil, fmt.Errorf("point must contain id, vector, or payload")
	}
	encoded, err := json.Marshal(logical)
	return json.RawMessage(encoded), err
}
func (adapter *Adapter) Apply(ctx context.Context, plan targets.Plan) error {
	for _, operation := range plan.Operations {
		current, err := adapter.Read(ctx, operation.Unit)
		if err != nil {
			return err
		}
		if operation.ExpectedExists {
			if !current.Exists || current.Fingerprint != operation.ExpectedFingerprint {
				return fmt.Errorf("target conflict for logical unit %s", operation.Unit)
			}
		} else if current.Exists {
			return fmt.Errorf("target conflict for logical unit %s", operation.Unit)
		}
		switch operation.Action {
		case ledger.ChangePut:
			var point any
			if err := json.Unmarshal(operation.Desired, &point); err != nil {
				return fmt.Errorf("decode Qdrant point %s: %w", operation.Unit, err)
			}
			if _, err := adapter.request(ctx, http.MethodPut, "/points?wait=true", map[string]any{"points": []any{point}}); err != nil {
				return err
			}
		case ledger.ChangeDelete:
			pointID := any(operation.Unit)
			if numericID, err := strconv.ParseUint(operation.Unit, 10, 64); err == nil {
				pointID = numericID
			}
			if current.Exists {
				var point struct {
					ID json.RawMessage `json:"id"`
				}
				if err := json.Unmarshal(current.Value, &point); err != nil || len(point.ID) == 0 {
					return fmt.Errorf("decode Qdrant point ID %s", operation.Unit)
				}
				pointID = point.ID
			}
			if _, err := adapter.request(ctx, http.MethodPost, "/points/delete?wait=true", map[string]any{"points": []any{pointID}}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported target operation %s", operation.Action)
		}
	}
	return nil
}
func (adapter *Adapter) Verify(ctx context.Context, plan targets.Plan) error {
	mismatches := make([]targets.VerificationMismatch, 0)
	for _, operation := range plan.Operations {
		value, err := adapter.Read(ctx, operation.Unit)
		if err != nil {
			return err
		}
		if operation.Action == ledger.ChangeDelete {
			if value.Exists {
				mismatches = append(mismatches, targets.VerificationMismatch{Unit: operation.Unit, Expected: operation.DesiredFingerprint, Observed: value.Fingerprint})
			}
			continue
		}
		if !value.Exists {
			mismatches = append(mismatches, targets.VerificationMismatch{Unit: operation.Unit, Expected: operation.DesiredFingerprint})
			continue
		}
		expected, err := normalizePoint(operation.Desired)
		if err != nil {
			return fmt.Errorf("normalize desired Qdrant point %s: %w", operation.Unit, err)
		}
		if !equivalentJSON(expected, value.Value, operation.TargetMetric) {
			mismatches = append(mismatches, targets.VerificationMismatch{Unit: operation.Unit, Expected: operation.DesiredFingerprint, Observed: value.Fingerprint})
		}
	}
	if len(mismatches) != 0 {
		return &targets.VerificationError{Mismatches: mismatches}
	}
	return nil
}
func (adapter *Adapter) Restore(ctx context.Context, plan targets.Plan) error {
	return adapter.Apply(ctx, plan)
}
func (adapter *Adapter) Capabilities() targets.Capabilities {
	return targets.Capabilities{AtomicApply: false, ExactPreview: false, ConditionalWrite: true, Batch: true, Restore: true}
}

func equivalentJSON(expected, actual json.RawMessage, metric string) bool {
	var left, right any
	if json.Unmarshal(expected, &left) != nil || json.Unmarshal(actual, &right) != nil {
		return false
	}
	leftPoint, leftOK := left.(map[string]any)
	rightPoint, rightOK := right.(map[string]any)
	if !leftOK || !rightOK || len(leftPoint) != len(rightPoint) {
		return false
	}
	for key, value := range leftPoint {
		if key == "vector" && strings.EqualFold(metric, "cosine") {
			if !equivalentCosineVector(value, rightPoint[key]) {
				return false
			}
			continue
		}
		if !equivalentValue(value, rightPoint[key]) {
			return false
		}
	}
	return true
}

func equivalentCosineVector(left, right any) bool {
	expected, leftOK := left.([]any)
	actual, rightOK := right.([]any)
	if !leftOK || !rightOK || len(expected) != len(actual) || len(expected) == 0 {
		return false
	}
	var dot, leftNorm, rightNorm float64
	for index := range expected {
		leftValue, leftOK := expected[index].(float64)
		rightValue, rightOK := actual[index].(float64)
		if !leftOK || !rightOK {
			return false
		}
		dot += leftValue * rightValue
		leftNorm += leftValue * leftValue
		rightNorm += rightValue * rightValue
	}
	if leftNorm == 0 || rightNorm == 0 {
		return leftNorm == rightNorm
	}
	return math.Abs(dot/math.Sqrt(leftNorm*rightNorm)-1) <= 1e-6
}

func equivalentValue(left, right any) bool {
	switch expected := left.(type) {
	case map[string]any:
		actual, ok := right.(map[string]any)
		if !ok || len(expected) != len(actual) {
			return false
		}
		for key, value := range expected {
			if !equivalentValue(value, actual[key]) {
				return false
			}
		}
		return true
	case []any:
		actual, ok := right.([]any)
		if !ok || len(expected) != len(actual) {
			return false
		}
		for index, value := range expected {
			if !equivalentValue(value, actual[index]) {
				return false
			}
		}
		return true
	case float64:
		actual, ok := right.(float64)
		if !ok {
			return false
		}
		return math.Abs(expected-actual) <= 1e-6*math.Max(1, math.Max(math.Abs(expected), math.Abs(actual)))
	default:
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
}
