package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func main() {
	dryRun := flag.Bool("dry-run", true, "preview inserts without writing model usage facts")
	cutoff := flag.Int64("cutoff", 0, "include historical records created/submitted at or before this Unix timestamp")
	batch := flag.String("batch", "", "backfill batch identifier")
	rollbackBatch := flag.String("rollback-batch", "", "delete model usage facts for this exact batch")
	confirmRollback := flag.Bool("confirm-rollback", false, "confirm rollback-batch deletion")
	flag.Parse()

	common.InitEnv()
	if err := model.InitDB(); err != nil {
		exitJSON(map[string]any{"success": false, "message": err.Error()})
	}
	if err := model.InitLogDB(); err != nil {
		exitJSON(map[string]any{"success": false, "message": err.Error()})
	}

	ctx := context.Background()
	if *rollbackBatch != "" {
		if !*confirmRollback {
			exitJSON(map[string]any{"success": false, "message": "--confirm-rollback is required"})
		}
		deleted, err := service.RollbackModelUsageBatch(ctx, *rollbackBatch)
		if err != nil {
			exitJSON(map[string]any{"success": false, "message": err.Error()})
		}
		writeJSON(map[string]any{"success": true, "deleted": deleted, "batch": *rollbackBatch})
		return
	}

	report, err := service.BackfillModelUsageEvents(ctx, service.BackfillOptions{
		DryRun: *dryRun,
		Cutoff: *cutoff,
		Batch:  *batch,
	})
	if err != nil {
		exitJSON(map[string]any{"success": false, "message": err.Error(), "report": report})
	}
	writeJSON(map[string]any{"success": true, "report": report})
}

func writeJSON(payload any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func exitJSON(payload any) {
	writeJSON(payload)
	os.Exit(1)
}

func init() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/model-usage-backfill.go --dry-run --cutoff <unix> --batch <name>")
	}
}
