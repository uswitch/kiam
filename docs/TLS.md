# Securing with TLS

Install the tools:

```
go get -u github.com/cloudflare/cfssl/cmd/...
```

## Generate certs

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
cfssl gencert -ca=ca.pem -ca-key=ca-key.pem server.json | cfssljson -bare agent
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
