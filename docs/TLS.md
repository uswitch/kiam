# TLS

Kiam is split into two processes:

* Agent. Responsible for receiving HTTP connections that Pods initiate to the metadata API (http://169.254.169.254). Credential requests are processed and anything else is forwarded to the AWS API. Communicates with the server via gRPC.
* Server. Runs the gRPC server that Agent uses to determine Pod roles and retrieve credentials.

To ensure that only agents and servers can communicate with each other they use mutual TLS authentication. These are not automatically generated so you'll need to create the certificates and store in secrets that only Server and Agent processes can access.

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
