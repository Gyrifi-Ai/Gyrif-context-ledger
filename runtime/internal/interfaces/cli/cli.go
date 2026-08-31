package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
)

func Run(ctx context.Context, args []string, application *engine.Engine, output io.Writer) (bool, error) {
	if handled, err := RunVersion(args, output); handled {
		return true, err
	}
	if len(args) == 0 {
		return false, nil
	}
	switch args[0] {
	case "doctor":
		ledgerCount := 0
		cursor := ""
		for {
			page, err := application.ListLedgers(ctx, engine.ListRequest{Limit: engine.MaxListLimit, Cursor: cursor})
			if err != nil {
				return true, err
			}
			ledgerCount += len(page.Items)
			if page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
		return true, json.NewEncoder(output).Encode(map[string]any{"status": "ok", "ledgers": ledgerCount, "inference": application.InferenceName()})
	default:
		return true, fmt.Errorf("unknown command %q", args[0])
	}
}

func RunVersion(args []string, output io.Writer) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if args[0] != "version" && args[0] != "--version" && args[0] != "-version" {
		return false, nil
	}
	_, err := fmt.Fprintln(output, buildinfo.String())
	return true, err
}
