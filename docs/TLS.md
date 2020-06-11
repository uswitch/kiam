# TLS

Kiam is split into two processes:

* Agent. Responsible for receiving HTTP connections that Pods initiate to the metadata API (http://169.254.169.254). Credential requests are processed and anything else is forwarded to the AWS API. Communicates with the server via gRPC.
* Server. Runs the gRPC server that Agent uses to determine Pod roles and retrieve credentials.

To ensure that only agents and servers can communicate with each other they use mutual TLS authentication. Therefore you need to set up some kind of PKI to create the coresponding certificates.

To do that you can (for testing purposes) [manually](#manually) create you PKI using tools like `cfssl`. If you have [cert-manager](https://github.com/jetstack/cert-manager) installed on your cluster, you can [use it](#cert-manager) to create your certificates. If you prefer any other way like [Vault](https://www.vaultproject.io/docs/secrets/pki/) that should be fine, too.

## Breaking Change in v3.0
In 3.0 a fix was made to the [TLS ServerName configuration](https://github.com/uswitch/kiam/pull/86). This changed the validation to only check the host (rather than the address). The configuration for the server certificate has been updated. 


## Manually

**If you generated your own certificates manually you will need to additively update them to include only the host to avoid breaking server/agent communication**.

### Install the helper tool
```
go get -u github.com/cloudflare/cfssl/cmd/...
```

### Generate certs
Sample cert json data provided in the [docs directory.](https://github.com/uswitch/kiam/tree/master/docs)

1. Create the Certificate Authority Cert and Key

```
cfssl gencert -initca ca.json | cfssljson -bare ca
```

2. Create Server pair

```
cfssl gencert -ca=ca.pem -ca-key=ca-key.pem server.json | cfssljson -bare server
```

3. Create Agent pair

```
cfssl gencert -ca=ca.pem -ca-key=ca-key.pem agent.json | cfssljson -bare agent
```

### Store in Kubernetes

```
kubectl create secret generic kiam-server-tls -n kube-system \
  --from-file=ca.pem \
  --from-file=server.pem \
  --from-file=server-key.pem
````

```
kubectl create secret generic kiam-agent-tls -n kube-system \
  --from-file=ca.pem \
  --from-file=agent.pem \
  --from-file=agent-key.pem
````

## Cert manager

You can use `cert-manager` to create a selfSigned issuer to create a CA and ca issuer for creating the required certs using that CA (note the following is only compatible with cert-manager version 0.11.0 or later):

```yaml
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: selfsigning-issuer
spec:
  selfSigned: {}

---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: example-ca
spec:
  secretName: ca-tls
  commonName: "my-ca"
  isCA: true
  issuerRef:
    name: selfsigning-issuer
  usages:
  - "any"

---
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: ca-issuer
spec:
  ca:
    secretName: ca-tls

---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: agent
spec:
  secretName: agent-tls
  commonName: agent
  issuerRef:
    name: ca-issuer
  usages:
  - "any"

---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: server
spec:
  secretName: server-tls
  issuerRef:
    name: ca-issuer
  usages:
  - "any"
  dnsNames:
  - "localhost"
  - "kiam-server"
  ipAddresses:
  - "127.0.0.1"
```
