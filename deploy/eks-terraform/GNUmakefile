EKS_CLUSTER=test
ASSUMED_ROLE=$(shell cat test.tfvars | grep assumed_role | cut -d\" -f2)
REGION=$(shell cat test.tfvars | grep region | cut -d\" -f2)
CREDFILE=.assumerolecreds
KUBECONFIG=kubeconfig-$(EKS_CLUSTER)
MANIFESTS_FOLDER=manifests_rendered
KUBECTL_CONTEXT=$(EKS_CLUSTER)

# kubectl command with common parameters
KUBECTL=kubectl --kubeconfig=$(KUBECONFIG) --context=$(KUBECTL_CONTEXT)

.PHONY: check-terraform-format
check-terraform-format:
	terraform fmt -check

.PHONY: tf-init
tf-init: check-terraform-format
	terraform init -reconfigure

.PHONY: tf-plan
tf-plan: tf-init
	terraform plan -var-file=$(EKS_CLUSTER).tfvars

.PHONY: tf-apply
tf-apply: tf-init
	terraform apply -auto-approve -var-file=$(EKS_CLUSTER).tfvars

.PHONY: tf-refresh
tf-refresh: tf-init
	terraform refresh -var-file=$(EKS_CLUSTER).tfvars > /dev/null

.PHONY: tf-clean
tf-clean:
	if [ -d .terraform ]; then\
		rm -rf .terraform;\
	fi;\

.PHONY: tf-destroy
tf-destroy: tf-init
	terraform destroy -var-file=$(EKS_CLUSTER).tfvars

# Gain access to the EKS cluster by updating the kubeconfig
.PHONY: access
access: 
	aws sts assume-role --role-arn $(ASSUMED_ROLE) --role-session-name update-kubeconfig --output text | tail -n1 > $(CREDFILE)
	AWS_ACCESS_KEY_ID=$$(cat $(CREDFILE) | cut -f2) AWS_SECRET_ACCESS_KEY=$$(cat $(CREDFILE) | cut -f4) AWS_SESSION_TOKEN=$$(cat $(CREDFILE) | cut -f5) aws eks update-kubeconfig --region="$(REGION)" --name="$(EKS_CLUSTER)" --role="$(ASSUMED_ROLE)" --alias="$(EKS_CLUSTER)" --kubeconfig $(KUBECONFIG)

# Bootstrap the cluster
.PHONY: bootstrap
bootstrap: access aws-auth kiam-ssl-setup kiam

.PHONY: aws-auth
	  $(KUBECTL) apply -f $(MANIFESTS_FOLDER)/aws-auth.yaml

# create kubernetes secrets for the connection between kiam-agent and *-server from generated files.
# This is required for kiam to work.
.PHONY: kiam-ssl-setup
kiam-ssl-setup:
	$(KUBECTL) apply -f $(MANIFESTS_FOLDER)/kiam-namespace.yaml
	cfssl gencert -initca kiam-assets/ca.json | cfssljson -bare ca
	cfssl gencert -ca=ca.pem -ca-key=ca-key.pem kiam-assets/server.json | cfssljson -bare server
	cfssl gencert -ca=ca.pem -ca-key=ca-key.pem kiam-assets/agent.json | cfssljson -bare agent
	$(KUBECTL) create secret generic kiam-server-tls -n kiam --from-file=ca.pem --from-file=server.pem --from-file=server-key.pem --dry-run -o yaml | $(KUBECTL) apply -f -
	$(KUBECTL) create secret generic kiam-agent-tls -n kiam --from-file=ca.pem --from-file=agent.pem --from-file=agent-key.pem --dry-run -o yaml | $(KUBECTL) apply -f -

# Deploy kiam
# First clusterroles and stuff, then server, then agent
# Note that the namespace is already applied in kiam-ssl-setup
.PHONY: kiam
kiam:
	$(KUBECTL) apply -f $(MANIFESTS_FOLDER)/kiam-base.yaml
	@for component in server agent; do \
	  $(KUBECTL) apply -f $(MANIFESTS_FOLDER)/kiam-$${component}.yaml; \
	  $(KUBECTL) rollout status daemonset kiam-$${component} --timeout 120s -n kiam ; \
	done

.PHONY: test-kiam
test-kiam:
	$(KUBECTL) apply -f $(MANIFESTS_FOLDER)/kiam-test.yaml

.PHONY: log-kiam
log-kiam:
	$(KUBECTL) logs -f -n kiam $$($(KUBECTL) get pods -n kiam | grep kiam-server | grep Running | cut -d' ' -f1 | head -n1)
