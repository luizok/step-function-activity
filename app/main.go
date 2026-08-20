package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

func main() {

	//TODO: Add retry
	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("sa-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	sfnClient := sfn.NewFromConfig(cfg)

	activityArn := os.Getenv("APPROVER_WORKER_ARN")
	workerName := "golang-worker-01"

	// aw := NewActivityWorker(workerName, activityArn, sfnClient, doSomeApproval)
	// aw.Start(ctx)

	dw := NewActivityWorker(workerName, activityArn, sfnClient, cancelProcessing)
	dw.Start(ctx)
}
