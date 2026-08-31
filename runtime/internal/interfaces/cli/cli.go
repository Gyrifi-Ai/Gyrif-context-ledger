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
	if len(args) == 0 {
		return false, nil
	}
	switch args[0] {
	case "doctor":
		ledgers, err := application.ListLedgers(ctx)
		if err != nil {
			return true, err
		}
		return true, json.NewEncoder(output).Encode(map[string]any{"status": "ok", "ledgers": len(ledgers), "inference": application.InferenceName()})
	case "version", "--version", "-version":
		_, err := fmt.Fprintln(output, buildinfo.String())
		return true, err
	default:
		return true, fmt.Errorf("unknown command %q", args[0])
	}
}
