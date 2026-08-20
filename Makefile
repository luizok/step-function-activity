infra-deploy:
	terraform fmt
	terraform validate
	terraform apply
