infra-deploy:
	terraform fmt
	terraform validate
	terraform apply

go:
	export APPROVER_WORKER_ARN=$(shell terraform output -json | jq -r '.approver_activity_arn.value') && \
	cd app/ && \
	go run main.go

start-execution:
	export STATE_MACHINE_ARN=$(shell terraform output -json | jq -r '.state_machine_arn.value') && \
	aws stepfunctions \
		start-execution \
		--state-machine-arn $$STATE_MACHINE_ARN \
		--input '{"datetime":"2026-08-19"}'
