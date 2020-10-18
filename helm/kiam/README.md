# kiam

Installs [kiam](https://github.com/uswitch/kiam) to integrate AWS IAM with Kubernetes.

## TL;DR;

```console
$ helm repo add uswitch https://uswitch.github.io/kiam-helm-charts/charts/
$ helm repo update
$ helm install uswitch/kiam
```

## Introduction

This chart bootstraps a [kiam](https://github.com/uswitch/kiam) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites
  - Kubernetes 1.8+ with Beta APIs enabled

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ helm install uswitch/kiam --name my-release
```

The command deploys kiam on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

## Key Configuration
The default helm configuration will probably not work out-of-the-box. You will most likely need to adjust the following:

* Kiam requires trusted SSL root certificates from the host system to be mounted into the kiam-server pod in order to be able to contact the AWS meta-data API. If the SSL cert directory on the host(s) you intend to run the kiam-server on does not match the default (`/usr/share/ca-certificates`), you will need to set the `server.sslCertHostPath` variable.
* Kiam will _not_ work without an appropriate iptables rule. As there are security & operational implications with making kiam responsible for inserting & removing the rule (see [#202](https://github.com/uswitch/kiam/issues/202) & [#253](https://github.com/uswitch/kiam/pull/253)), the `agent.host.iptables` parameter is set to `false` by default. Either configure the iptables rule separately from Kiam, or use `--set agent.host.iptables=true`.

### Adding `iptables` rule separately
The most secure way to configure `kiam` is to install its `iptables` rule before starting Docker on the host, and leave it in place. This way when `kiam` is not able to serve credentials, any clients attempting to refresh credentials will get an error, and they should continue to use their cached credentials while periodically retrying to refresh them. Likewise clients attempting to get credentials for the first time will get nothing, rather than get the credentials associated with the host they are running on.

You can [read the code](https://github.com/uswitch/kiam/blob/master/cmd/kiam/iptables.go), but the definitive way to see the `iptables` rule `kiam` installs is to `--set agent.host.iptables=true`, deploy the agent, find the pod name for an agent pod and exec into it:
```bash
kiam_pod=$(kubectl get pods | grep kiam-agent | awk '{print $1}' | head -n 1)
kubectl exec -it $kiam_pod -- iptables -t nat -S PREROUTING | grep 169.254.169.254/32
```
This will print out the arguments to follow `iptables -t nat` needed to install the rule. However, the arguments will include the IP address of the machine, so you cannot just copy and paste them verbatim. You can, however, use it to double-check you have the correct rule installed. 

This command works on `debian` hosts on AWS when using `calico` networking:
```bash
/sbin/iptables -t nat -A PREROUTING -d 169.254.169.254/32 \
        -i cali+ -p tcp -m tcp --dport 80 -j DNAT \
        --to-destination $(curl -s http://169.254.169.254/latest/meta-data/local-ipv4):8181
```
Replace `cali+` with the CNI interface name you are passing to `kiam-agent` as `--host-interface` and if you are using exclamation point to invert it, e.g. `!eth0` be sure to escape the exclamation point so it is not interpreted by the shell. 

It is safe to have the rule installed twice, so you can test your installation by installing the rule once via the `kiam-agent` and once some other way, in which case you should see the same rule installed twice when you print out the rules. If the 2 rules are identical, then you have the install correct and can revert `agent.host.iptables` to false. 


## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## TLS Certificates

`Note:` The chart generates a self signed TLS certificate by default, the values mentioned below only need changing if you plan to use your own TLS certificates.

If needed, you can create TLS certificates and private keys as described [here](https://github.com/uswitch/kiam/blob/master/docs/TLS.md).

> **Tip**: The `hosts` field in the kiam server certificate should include the value _release-name_-server:_server-service-port_, e.g. `my-release-server:443`</br>
> If you don't include the exact hostname used by the kiam agent to connect to the server, you'll see a warning (which is really an error) in the agent logs similar to the following, and your pods will fail to obtain credentials:
```json
{"level":"warning","msg":"error finding role for pod: rpc error: code = Unavailable desc = there is no connection available","pod.ip":"100.120.0.2","time":"2018-05-24T04:11:25Z"}
```

Define values `agent.tlsFiles.ca`, `agent.tlsFiles.cert`, `agent.tlsFiles.key`, `server.tlsFiles.ca`, `server.tlsFiles.cert` and `server.tlsFiles.key` to be the base64-encoded contents (.e.g. using the `base64` command) of the generated PEM files.
For example

```yaml
agent:
  tlsFiles:
    key: LS0tL...
    cert: LS0tL...
    ca: LS0tL...

server:
  tlsFiles:
    key: LS0tL...
    cert: LS0tL...
    ca: LS0tL...
```

Define secret name values `agent.tlsSecret` and `server.tlsSecret` if TLS certificates secrets have already created instead of `tlsFiles`.

```yaml
agent:
  tlsSecret: kiam-agent-tls

server:
  tlsSecret: kiam-server-tls
```
Define TLS certificate names to use in kiam command line arguments as follows.
```yaml
agent:
  tlsCerts:
    certFileName: cert
    keyFileName: key
    caFileName: ca

server:
  tlsCerts:
    certFileName: cert
    keyFileName: key
    caFileName: ca
```

## SELinux Options

For SELinux enabled systems, such as OpenShift on RHEL, you may need to
apply special SELinux labels to the agent and/or server processes.

In order for the agent to access `/run/xtable.lock`, access to the
`iptables_var_run_t` type is required. This can be achieved by
giving the process `spc_t` rather than `container_t`:

```yaml

agent:
  seLinuxOptions:
    user: system_u
    role: system_r
    type: spc_t
    level: s0
```

Similarly, in order for the server to access the hosts CA certificates,
such as `/etc/pki/ca-trust/extracted/pem`, it will need access to `cert_t`
which can also be granted via `spc_t`:

```yaml

server:
  sslCertHostPath: /etc/pki/ca-trust/extracted/pem
  seLinuxOptions:
    user: system_u
    role: system_r
    type: spc_t
    level: s0
```

## Configuration

The following table lists the configurable parameters of the kiam chart and their default values.

| Parameter                                   | Description                                                                                  | Default                      |
| ------------------------------------------- | -------------------------------------------------------------------------------------------- | ---------------------------- |
| `agent.enabled`                             | If true, create agent                                                                        | `true`                       |
| `agent.name`                                | Agent container name                                                                         | `agent`                      |
| `agent.image.repository`                    | Agent image                                                                                  | `quay.io/uswitch/kiam`       |
| `agent.image.tag`                           | Agent image tag                                                                              | `v3.6`                       |
| `agent.image.pullPolicy`                    | Agent image pull policy                                                                      | `IfNotPresent`               |
| `agent.dnsPolicy`                           | Agent pod DNS policy                                                                         | `ClusterFirstWithHostNet`    |
| `agent.allowRouteRegexp`                    | Agent metadata proxy server only allows accesses to paths matching this regexp               | `{}`                         |
| `agent.extraArgs`                           | Additional agent container arguments                                                         | `{}`                         |
| `agent.extraEnv`                            | Additional agent container environment variables                                             | `{}`                         |
| `agent.extraHostPathMounts`                 | Additional agent container hostPath mounts                                                   | `[]`                         |
| `agent.initContainers`                      | Agent initContainers                                                                         | `[]`                         |
| `agent.gatewayTimeoutCreation`              | Agent's timeout when creating the kiam gateway                                               | `1s`                         |
| `agent.keepaliveParams.time`                | gRPC keepalive time                                                                          | `10s`                        |
| `agent.keepaliveParams.timeout`             | gRPC keepalive timeout                                                                       | `2s`                         |
| `agent.keepaliveParams.permitWithoutStream` | gRPC keepalive ping even with no RPC                                                         | `false`                      |
| `agent.host.ip`                             | IP address of host                                                                           | `$(HOST_IP)`                 |
| `agent.host.iptables`                       | Add iptables rule                                                                            | `false`                      |
| `agent.host.interface`                      | Agent's host interface for proxying AWS metadata                                             | `cali+`                      |
| `agent.host.port`                           | Agent's listening port                                                                       | `8181`                       |
| `agent.log.jsonOutput`                      | Whether or not to output agent log in JSON format                                            | `true`                       |
| `agent.log.level`                           | Agent log level (`debug`, `info`, `warn` or `error`)                                         | `info`                       |
| `agent.deepLivenessProbe`                   | Fail liveness probe if the server is not accessible                                          | `false`                      |
| `agent.nodeSelector`                        | Node labels for agent pod assignment                                                         | `{}`                         |
| `agent.prometheus.port`                     | Agent Prometheus metrics port                                                                | `9620`                       |
| `agent.prometheus.scrape`                   | Whether or not Prometheus metrics for the agent should be scraped                            | `true`                       |
| `agent.prometheus.syncInterval`             | Agent Prometheus synchronization interval                                                    | `5s`                         |
| `agent.prometheus.servicemonitor.enabled`   | Whether servicemonitor resource should be deployed for the agent                             | `false`                      |
| `agent.prometheus.servicemonitor.path`      | Agent prometheus scrape path                                                                 | `/metrics`                   |
| `agent.prometheus.servicemonitor.interval`  | Agent prometheus scrape interval from servicemonitor                                         | `10s`                        |
| `agent.prometheus.servicemonitor.labels`    | Custom labels for agent servicemonitor                                                       | `{}`                         |
| `agent.podAnnotations`                      | Annotations to be added to agent pods                                                        | `{}`                         |
| `agent.podLabels`                           | Labels to be added to agent pods                                                             | `{}`                         |
| `agent.priorityClassName`                   | Agent pods priority class name                                                               | `""`                         |
| `agent.resources`                           | Agent container resources                                                                    | `{}`                         |
| `agent.seLinuxOptions`                      | SELinux labels to be added to the agent container process                                    | `{}`                         |
| `agent.serviceAnnotations`                  | Annotations to be added to agent service                                                     | `{}`                         |
| `agent.serviceLabels`                       | Labels to be added to agent service                                                          | `{}`                         |
| `agent.tlsSecret`                           | Secret name for the agent's TLS certificates                                                 | `null`                       |
| `agent.tlsFiles.ca`                         | Base64 encoded string for the agent's CA certificate(s)                                      | `null`                       |
| `agent.tlsFiles.cert`                       | Base64 encoded strings for the agent's certificate                                           | `null`                       |
| `agent.tlsFiles.key`                        | Base64 encoded strings for the agent's private key                                           | `null`                       |
| `agent.tolerations`                         | Tolerations to be applied to agent pods                                                      | `[]`                         |
| `agent.affinity`                            | Node affinity for pod assignment                                                             | `{}`                         |
| `agent.updateStrategy`                      | Strategy for agent DaemonSet updates (requires Kubernetes 1.6+)                              | `OnDelete`                   |
| `agent.livenessProbe.initialDelaySeconds`   | Delay before liveness probe is initiated                                                     | 3                            |
| `agent.livenessProbe.periodSeconds`         | How often to perform the probe                                                               | 3                            |
| `agent.livenessProbe.timeoutSeconds`        | When the probe times out                                                                     | 1                            |
| `agent.livenessProbe.successThreshold`      | Minimum consecutive successes for the probe to be considered successful after having failed. | 1                            |
| `agent.livenessProbe.failureThreshold`      | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3                            |
| `server.enabled`                            | If true, create server                                                                       | `true`                       |
| `server.name`                               | Server container name                                                                        | `server`                     |
| `server.gatewayTimeoutCreation`             | Server's timeout when creating the kiam gateway                                              | `1s`                         |
| `server.image.repository`                   | Server image                                                                                 | `quay.io/uswitch/kiam`       |
| `server.image.tag`                          | Server image tag                                                                             | `v3.6`                       |
| `server.image.pullPolicy`                   | Server image pull policy                                                                     | `Always`                     |
| `server.assumeRoleArn`                      | IAM role for the server to assume before processing requests                                 | `null`                       |
| `server.cache.syncInterval`                 | Pod cache synchronization interval                                                           | `1m`                         |
| `server.extraArgs`                          | Additional server container arguments                                                        | `{}`                         |
| `server.extraEnv`                           | Additional server container environment variables                                            | `{}`                         |
| `server.sslCertHostPath`                    | Path to SSL certs on host machinee                                                           | `/usr/share/ca-certificates` |
| `server.extraHostPathMounts`                | Additional server container hostPath mounts                                                  | `[]`                         |
| `server.initContainers`                     | Server initContainers                                                                        | `[]`                         |
| `server.log.jsonOutput`                     | Whether or not to output server log in JSON format                                           | `true`                       |
| `server.log.level`                          | Server log level (`debug`, `info`, `warn` or `error`)                                        | `info`                       |
| `server.nodeSelector`                       | Node labels for server pod assignment                                                        | `{}`                         |
| `server.prometheus.port`                    | Server Prometheus metrics port                                                               | `9620`                       |
| `server.prometheus.scrape`                  | Whether or not Prometheus metrics for the server should be scraped                           | `true`                       |
| `server.prometheus.syncInterval`            | Server Prometheus synchronization interval                                                   | `5s`                         |
| `server.prometheus.servicemonitor.enabled`  | Whether servicemonitor resource should be deployed for the server                            | `false`                      |
| `server.prometheus.servicemonitor.path`     | Server prometheus scrape path                                                                | `/metrics`                   |
| `server.prometheus.servicemonitor.interval` | Server prometheus scrape interval from servicemonitor                                        | `10s`                        |
| `server.prometheus.servicemonitor.labels`   | Custom labels for server servicemonitor                                                      | `{}`                         |
| `server.podAnnotations`                     | Annotations to be added to server pods                                                       | `{}`                         |
| `server.podLabels`                          | Labels to be added to server pods                                                            | `{}`                         |
| `server.probes.serverAddress`               | Address that readyness and liveness probes will hit                                          | `127.0.0.1`                  |
| `server.priorityClassName`                  | Server pods priority class name                                                              | `""`                         |
| `server.resources`                          | Server container resources                                                                   | `{}`                         |
| `server.roleBaseArn`                        | Base ARN for IAM roles. If not specified use EC2 metadata service to detect ARN prefix       | `null`                       |
| `server.seLinuxOptions`                     | SELinux labels to be added to the server container process                                   | `{}`                         |
| `server.sessionDuration`                    | Session duration for STS tokens generated by the server                                      | `15m`                        |
| `server.serviceAnnotations`                 | Annotations to be added to server service                                                    | `{}`                         |
| `server.serviceLabels`                      | Labels to be added to server service                                                         | `{}`                         |
| `server.service.port`                       | Server service port                                                                          | `443`                        |
| `server.service.targetPort`                 | Server service target port                                                                   | `443`                        |
| `server.tlsSecret`                          | Secret name for the server's TLS certificates                                                | `null`                       |
| `server.tlsFiles.ca`                        | Base64 encoded string for the server's CA certificate(s)                                     | `null`                       |
| `server.tlsFiles.cert`                      | Base64 encoded strings for the server's certificate                                          | `null`                       |
| `server.tlsFiles.key`                       | Base64 encoded strings for the server's private key                                          | `null`                       |
| `server.tolerations`                        | Tolerations to be applied to server pods                                                     | `[]`                         |
| `server.affinity`                           | Node affinity for pod assignment                                                             | `{}`                         |
| `server.updateStrategy`                     | Strategy for server DaemonSet updates (requires Kubernetes 1.6+)                             | `OnDelete`                   |
| `server.useHostNetwork`                     | If true, use hostNetwork on server to bypass agent iptable rules                             | `false`                      |
| `server.livenessProbe.initialDelaySeconds`  | Delay before liveness probe is initiated                                                     | 10                           |
| `server.livenessProbe.periodSeconds`        | How often to perform the probe                                                               | 10                           |
| `server.livenessProbe.timeoutSeconds`       | When the probe times out                                                                     | 10                           |
| `server.livenessProbe.successThreshold`     | Minimum consecutive successes for the probe to be considered successful after having failed. | 1                            |
| `server.livenessProbe.failureThreshold`     | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3                            |
| `server.readinessProbe.initialDelaySeconds` | Delay before readiness probe is initiated                                                    | 10                           |
| `server.readinessProbe.periodSeconds`       | How often to perform the probe                                                               | 10                           |
| `server.readinessProbe.timeoutSeconds`      | When the probe times out                                                                     | 10                           |
| `server.readinessProbe.successThreshold`    | Minimum consecutive successes for the probe to be considered successful after having failed. | 1                            |
| `server.readinessProbe.failureThreshold`    | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3                            |
| `rbac.create`                               | If `true`, create & use RBAC resources                                                       | `true`                       |
| `psp.create`                                | If `true`, create Pod Security Policies for the agent and server when enabled                | `false`                      |
| `imagePullSecrets`                          | The name of the secret to use if pulling from a private registry                             | `nil`                        |
| `serviceAccounts.agent.create`              | If true, create the agent service account                                                    | `true`                       |
| `serviceAccounts.agent.name`                | Name of the agent service account to use or create                                           | `{{ kiam.agent.fullname }}`  |
| `serviceAccounts.server.create`             | If true, create the server service account                                                   | `true`                       |
| `serviceAccounts.server.name`               | Name of the server service account to use or create                                          | `{{ kiam.server.fullname }}` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
$ helm install uswitch/kiam --name my-release \
  --set=extraArgs.base-role-arn=arn:aws:iam::0123456789:role/,extraArgs.default-role=kube2iam-default,host.iptables=true,host.interface=cbr0
```

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install uswitch/kiam --name my-release -f values.yaml
```

> **Tip**: You can use the default [values.yaml](values.yaml)
