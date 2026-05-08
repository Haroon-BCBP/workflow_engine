package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	engine "github.com/Haroon-BCBP/workflow_engine/internal/workflow"
)

func main() {
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")

	tc, err := client.Dial(client.Options{HostPort: temporalHost})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer tc.Close()

	w := worker.New(tc, engine.TaskQueue, worker.Options{})

	w.RegisterWorkflow(engine.DSLWorkflow)

	acts := &engine.Activities{}
	w.RegisterActivity(acts.StageStartedActivity)
	w.RegisterActivity(acts.SaveCommentActivity)

	log.Printf("Worker started on task queue: %s", engine.TaskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
