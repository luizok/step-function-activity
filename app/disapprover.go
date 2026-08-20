package main

import (
	"os"
	"time"
)

type DisapproverWorkerInput struct {
	JobRunId           string `json:"job_run_id"`
	ProcessedPartition string `json:"processed_partition"`
}

type DisapproverWorkerOutput struct {
	WorkerNameF string `json:"worker_name"`
	TaskTokenF  string `json:"task_token"`
	HasErrorF   bool   `json:"has_error"`
	RejectedBy  string `json:"reject_by"`
	DisapproverWorkerInput
}

func (awo DisapproverWorkerOutput) HasError() bool {
	return awo.HasErrorF
}

func (awo DisapproverWorkerOutput) TaskToken() string {
	return awo.TaskTokenF
}

func (awo *DisapproverWorkerOutput) SetTaskToken(taskToken string) {
	awo.TaskTokenF = taskToken
}

func (awo DisapproverWorkerOutput) WorkerName() string {
	return awo.WorkerNameF
}

func (awo *DisapproverWorkerOutput) SetWorkerName(workerName string) {
	awo.WorkerNameF = workerName
}

func cancelProcessing(input DisapproverWorkerInput) (*DisapproverWorkerOutput, error) {

	output := new(DisapproverWorkerOutput)
	output.JobRunId = input.JobRunId
	output.ProcessedPartition = input.ProcessedPartition

	output.HasErrorF = false

	rejectedBy, _ := os.Hostname()
	output.HasErrorF = true
	output.RejectedBy = rejectedBy

	time.Sleep(5 * time.Second)

	return output, nil
}
