infra-deploy:
	terraform fmt
	terraform validate
	terraform apply

go:
	export APPROVER_WORKER_ARN=$(shell terraform output -json | jq -r '.approver_activity_arn.value') && \
	cd app/ && \
	go run main.go
