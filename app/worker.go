package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type ActivityOutputer interface {
	SetWorkerName(string)
	WorkerName() string
	SetTaskToken(string)
	TaskToken() string
	HasError() bool
}
type ActivityWorkerFunc[I any, O ActivityOutputer] func(I) (O, error)

type ActivityWorker[I any, O ActivityOutputer] struct {
	workerName  string
	activityArn string
	sfnClient   *sfn.Client
	doFunc      ActivityWorkerFunc[I, O]
}

func NewActivityWorker[I any, O ActivityOutputer](workerName, activityArn string, sfnClient *sfn.Client, doFunc ActivityWorkerFunc[I, O]) *ActivityWorker[I, O] {
	return &ActivityWorker[I, O]{
		workerName:  workerName,
		activityArn: activityArn,
		sfnClient:   sfnClient,
		doFunc:      doFunc,
	}
}

func (aw *ActivityWorker[I, O]) Start(ctx context.Context) {
	fmt.Println("Polling for an activity task...")
	for {
		output, err := aw.sfnClient.GetActivityTask(ctx, &sfn.GetActivityTaskInput{
			ActivityArn: aws.String(aw.activityArn),
			WorkerName:  aws.String(aw.workerName), // Used for execution history logs
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

		var inputData I
		json.Unmarshal([]byte(*output.Input), &inputData)
		fmt.Printf("JSON Input Data: %+v\n", inputData)

		outputData, err := aw.doFunc(inputData)
		if err != nil {
			fmt.Print(err)
		}

		outputData.SetWorkerName(aw.workerName)
		outputData.SetTaskToken(*output.TaskToken)
		outputDataJson, err := json.Marshal(&outputData)
		if err != nil {
			fmt.Print(err)
		}

		fmt.Printf("JSON Output Data: %+v\n", outputData)

		if !outputData.HasError() {
			_, err := aw.sfnClient.SendTaskSuccess(
				ctx,
				&sfn.SendTaskSuccessInput{
					TaskToken: aws.String(outputData.TaskToken()),
					Output:    aws.String(string(outputDataJson)),
				},
			)

			if err != nil {
				fmt.Print(err)
			}

			continue
		}

		_, err = aw.sfnClient.SendTaskFailure(
			ctx,
			&sfn.SendTaskFailureInput{
				TaskToken: aws.String(outputData.TaskToken()),
				Cause:     aws.String(string(outputDataJson)),
			},
		)

		if err != nil {
			fmt.Print(err)
		}
	}
}
