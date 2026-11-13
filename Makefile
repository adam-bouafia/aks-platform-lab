.PHONY: tools-check

tools-check:
	@git --version
	@gh --version
	@go version
	@docker --version
	@kind version
	@kubectl version --client
	@helm version
	@terraform version
	@az version
	@kubelogin --version
	@trivy --version
	@cosign version
	@hadolint --version
	@tflint --version
