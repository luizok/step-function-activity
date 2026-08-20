package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type ApproverWorkerInput struct {
	JobRunId           string `json:"job_run_id"`
	ProcessedPartition string `json:"processed_partition"`
}

type ApproverWorkerOutput struct {
	TaskToken  string `json:"task_token"`
	HasError   bool   `json:"has_error"`
	ApprovedBy string `json:"approved_by"`
	NativeOS   string `json:"native_os"`
	NativeArch string `json:"native_arch"`
	ApproverWorkerInput
}

func doSomeApproval(taskToken string, input ApproverWorkerInput) (*ApproverWorkerOutput, error) {

	output := new(ApproverWorkerOutput)
	output.JobRunId = input.JobRunId
	output.ProcessedPartition = input.ProcessedPartition
	output.TaskToken = taskToken

	output.HasError = false
	output.NativeOS = runtime.GOOS
	output.NativeArch = runtime.GOARCH

	approvedBy, err := os.Hostname()
	if err != nil {
		output.HasError = true
		return output, err
	}
	output.ApprovedBy = approvedBy

	time.Sleep(5 * time.Second)

	return output, nil
}

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

	fmt.Println("Polling for an activity task...")

	for {
		output, err := sfnClient.GetActivityTask(ctx, &sfn.GetActivityTaskInput{
			ActivityArn: aws.String(activityArn),
			WorkerName:  aws.String(workerName), // Used for execution history logs
		})
		if err != nil {
			log.Fatalf("failed to get activity task: %v", err)
		}

		if output.TaskToken == nil || *output.TaskToken == "" {
			fmt.Println("No tasks available. Polling timed out (60s).")
			return
		}

		// 6. Process the retrieved task payload
		fmt.Printf("Task received!\n")
		fmt.Printf("Task Token: %s\n", *output.TaskToken)

		var inputData ApproverWorkerInput
		json.Unmarshal([]byte(*output.Input), &inputData)
		fmt.Printf("JSON Input Data: %+v\n", inputData)

		outputData, err := doSomeApproval(*output.TaskToken, inputData)
		outputDataJson, err := json.Marshal(&outputData)
		if err != nil {
			fmt.Print(err)
		}

		fmt.Printf("JSON Output Data: %+v\n", outputData)

		if !outputData.HasError {
			_, err := sfnClient.SendTaskSuccess(
				ctx,
				&sfn.SendTaskSuccessInput{
					TaskToken: aws.String(*output.TaskToken),
					Output:    aws.String(string(outputDataJson)),
				},
			)

			if err != nil {
				fmt.Print(err)
			}

			continue
		}

		_, err = sfnClient.SendTaskFailure(
			ctx,
			&sfn.SendTaskFailureInput{
				TaskToken: aws.String(*output.TaskToken),
				Cause:     aws.String(string(outputDataJson)),
			},
		)

		if err != nil {
			fmt.Print(err)
		}
	}
}
