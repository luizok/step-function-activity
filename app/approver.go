package main

import (
	"os"
	"runtime"
	"time"
)

type ApproverWorkerInput struct {
	JobRunId           string `json:"job_run_id"`
	ProcessedPartition string `json:"processed_partition"`
}

type ApproverWorkerOutput struct {
	WorkerNameF string `json:"worker_name"`
	TaskTokenF  string `json:"task_token"`
	HasErrorF   bool   `json:"has_error"`
	ApprovedBy  string `json:"approved_by"`
	NativeOS    string `json:"native_os"`
	NativeArch  string `json:"native_arch"`
	ApproverWorkerInput
}

func (awo ApproverWorkerOutput) HasError() bool {
	return awo.HasErrorF
}

func (awo ApproverWorkerOutput) TaskToken() string {
	return awo.TaskTokenF
}

func (awo *ApproverWorkerOutput) SetTaskToken(taskToken string) {
	awo.TaskTokenF = taskToken
}

func (awo ApproverWorkerOutput) WorkerName() string {
	return awo.WorkerNameF
}

func (awo *ApproverWorkerOutput) SetWorkerName(workerName string) {
	awo.WorkerNameF = workerName
}

func doSomeApproval(input ApproverWorkerInput) (*ApproverWorkerOutput, error) {

	output := new(ApproverWorkerOutput)
	output.JobRunId = input.JobRunId
	output.ProcessedPartition = input.ProcessedPartition

	output.HasErrorF = false
	output.NativeOS = runtime.GOOS
	output.NativeArch = runtime.GOARCH

	approvedBy, err := os.Hostname()
	if err != nil {
		output.HasErrorF = true
		return output, err
	}
	output.ApprovedBy = approvedBy

	time.Sleep(5 * time.Second)

	return output, nil
}
