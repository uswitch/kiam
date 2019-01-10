# Kiam Prometheus Setup

Kiam server and agent exposes prometheus metrics at `/metrics` endpoint in the default `9620` port.

Create a ServiceMonitor which scrapes the metrics from the server and agent.

```
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata: 
    name: kiam-metrics
    namespace: kube-system
    labels: 
      prometheus: kube-prometheus
spec: 
  jobLabel: kiam-metrics
  endpoints: 
  - interval: 15s
    port: metrics
  selector: 
    matchLabels:
      app: kiam
      release: kiam
  namespaceSelector:
    matchNames:
    - kube-system
```

By default, the service of kiam server does not expose the `9620` port, this requires to edit the service to have the `metrics` port.

```
apiVersion: v1
kind: Service
metadata:
  labels:
    app: kiam
    component: server
  name: kiam-server
  namespace: kube-system
spec:
  clusterIP: None
  ports:
  - name: grpc
    port: 443
    protocol: TCP
    targetPort: 443
  - name: metrics
    port: 9620
    protocol: TCP
    targetPort: 9620
  selector:
    app: kiam
    component: server
    release: kiam
  sessionAffinity: None
  type: ClusterIP
```

To get the metrics from the kiam agent, modify the kiam agent daemon set to have the container port exposed

```
    ports:
    - containerPort: 9620
      name: metrics
```

Or create a service for kiam-agent pods

```
apiVersion: v1
kind: Service
metadata:
  labels:
    app: kiam
    component: agent
  name: kiam-agent
  namespace: kube-system
spec:
  clusterIP: None
  ports:
  - name: metrics
    port: 9620
    protocol: TCP
    targetPort: 9620
  selector:
    app: kiam
    component: agent
    release: kiam
  sessionAffinity: None
  type: ClusterIP
```