# Sample deployment of kiam on AWS EKS
This is a sample deployment of kiam on Amazon EKS with minimal dependencies.

## Dependencies

- terraform for creating the resources on AWS
- aws cli
- aws-iam-authenticator for authenticating to the cluster
- kubectl
- cfssl
- basic linux tools

## Spinning up the cluster

First, fill in `test.tvars` You can use `make tf-plan` to test your terraform deployment. The makefile will reject code that does not conform to `terraform fmt`.

If your tf-plan looks good, use `make fulldeploy` to deploy the cluster, configure kiam, and run a small test pod on it.
