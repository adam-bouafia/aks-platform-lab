# Commands log

## 2026-11-13 Phase 0 - Setup (1am)

### Repo

- Created the GitHub repo and pushed the first commit (`Init repo with readme and .gitignore`).
- `git@github.com:adam-bouafia/aks-platform-lab.git`

### Tools check

Added a `tools-check` using `Makefile`. It prints the version of every tool the project needs.

```
make tools-check
```

| Tool | Version | Version command | Used for |
|---|---|---|---|
| git | 2.56.0 | `git --version` | version control |
| gh | 2.97.0 | `gh --version` | GitHub CLI |
| go | 1.26.10 | `go version` | ledger-api (Phase 1) |
| docker | 29.9.3 | `docker --version` | build and run images |
| kind | 0.33.0 | `kind version` | local Kubernetes cluster |
| kubectl | 1.37.2 | `kubectl version --client` | talk to clusters |
| helm | 4.2.2 | `helm version` | Helm chart (Phase 4) |
| terraform | 1.17.2 | `terraform version` | Azure infrastructure (Phase 3) |
| az | 2.91.0 | `az version` | Azure CLI |
| kubelogin | 0.2.22 | `kubelogin --version` | Entra ID login for kubectl |
| trivy | 0.69.3 | `trivy --version` | image and IaC scanning |
| cosign | 3.1.4 | `cosign version` | image signing (Phase 2b) |
| hadolint | 2.15.1 | `hadolint --version` | Dockerfile linter |
| tflint | 0.66.0 | `tflint --version` | Terraform linter |

Notes:
- Makefile recipe lines must start with a Tab, not spaces (otherwise `missing separator`).
- `kubectl version --client` only checks the local binary. Without `--client` it tries to reach a cluster.
- kubectl must be within one minor version of the cluster.
