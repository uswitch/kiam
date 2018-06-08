# TLS

Kiam is split into two processes:

* Agent. Responsible for receiving HTTP connections that Pods initiate to the metadata API (http://169.254.169.254). Credential requests are processed and anything else is forwarded to the AWS API. Communicates with the server via gRPC.
* Server. Runs the gRPC server that Agent uses to determine Pod roles and retrieve credentials.

To ensure that only agents and servers can communicate with each other they use mutual TLS authentication. These are not automatically generated so you'll need to create the certificates and store in secrets that only Server and Agent processes can access.

## Breaking Change in v3.0
In 3.0 a fix was made to the [TLS ServerName configuration](https://github.com/uswitch/kiam/pull/86). This changed the validation to only check the host (rather than the address). The configuration for the server certificate has been updated. 

**If you generated your own certificates you will need to additively update them to include only the host to avoid breaking server/agent communication**.

## Install the helper tool
```
go get -u github.com/cloudflare/cfssl/cmd/...
```

## Generate certs
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

## Store in Kubernetes

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
