# kiam

[![Docker Repository on Quay](https://quay.io/repository/uswitch/kiam/status "Docker Repository on Quay")](https://quay.io/repository/uswitch/kiam)

kiam runs as an agent on each node in your Kubernetes cluster and allows cluster users to associate IAM roles to Pods.

Docker images are available at [https://quay.io/repository/uswitch/kiam](https://quay.io/repository/uswitch/kiam).

[@jbeda](https://github.com/jbeda) recorded an excellent TGIK video covering IAM configuration and Kiam installation:

* [TGI Kubernetes 070: Assuming AWS roles with kube2iam/kiam](https://www.youtube.com/watch?v=vgs3Af_ew3c)

For more information about Kiam's origin, design and performance in our production clusters:

* [Iterating for Security and Reliability](https://medium.com/@pingles/kiam-iterating-for-security-and-reliability-5e793ab93ec3)

## Support

If you come across issues please raise them against the GitHub project. Please include as much detail as you can, including relevant logs, version information etc.

We have a [#kiam Slack channel](https://kubernetes.slack.com/messages/CBQLKVABH/) in the main Kubernetes community Slack so you can also chat with other users and developers.

## Features

* No client SDK modifications are needed: Kiam intercepts Metadata API requests.
* Separated Agent and Server processes. Allows user workloads to run on nodes without `sts:AssumeRole` permissions to enhance cluster security.
* Denies access to all other AWS Metadata API paths by default (but can be configured via flag)
* AWS credentials are prefetched to allow fast responses (and avoid problems with races between Pods requesting credentials and the Kubernetes client caches being aware of the Pod)
* Multi-account IAM support. Pods can assume roles from any AWS account assuming trust relationships permit it
* [Prometheus metrics](docs/METRICS.md)
* Uses the Kubernetes Events API to record IAM errors against the Pod so that cluster users can more readily diagnose IAM problems (via `kubectl describe pod ...`)
* Text and JSON log formats
* Optional regional STS endpoint support

## Overview
From the [AWS documentation on IAM roles](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html):

> [a] role is similar to a user, in that it is an AWS identity with permission policies that determine what the identity can and cannot do in AWS. However, instead of being uniquely associated with one person, a role is intended to be assumable by anyone who needs it. Also, a role does not have any credentials (password or access keys) associated with it. Instead, if a user is assigned to a role, access keys are created dynamically and provided to the user.

kiam uses an annotation added to a `Pod` to indicate which role should be assumed. For example:

```yaml
kind: Pod
metadata:
  name: foo
  namespace: iam-example
  annotations:
    iam.amazonaws.com/role: reportingdb-reader
```

Further, all namespaces must also have an annotation with a regular expression expressing which roles are permitted to be assumed within that namespace. **Without the namespace annotation the pod will be unable to assume any roles.**

```yaml
kind: Namespace
metadata:
  name: iam-example
  annotations:
    iam.amazonaws.com/permitted: ".*"
```

When your process starts an AWS SDK library will normally use a chain of credential providers (environment variables, instance metadata, config files etc.) to determine which credentials to use. kiam intercepts the metadata requests and uses the [Security Token Service](http://docs.aws.amazon.com/STS/latest/APIReference/Welcome.html) to retrieve temporary role credentials.

## Deploying to Kubernetes
Please see the `deploy` directory for example manifests for deploying to Kubernetes.

TLS assets must be created to mutually authenticate the agents and server processes; notes are in [docs/TLS.md](docs/TLS.md).

Please also make note of how to configure IAM in your AWS account; notes in [docs/IAM.md](docs/IAM.md).

### Helm

We maintain and host Helm charts for Kiam, which are automatically packaged upon merging chart changes to the master branch in this repo. The charts can be found in the repo [here](https://github.com/uswitch/kiam/tree/master/helm/kiam).

The charts can easily be installed by adding the Kiam Helm repository and running a `helm install`:

```console
$ helm repo add uswitch https://uswitch.github.io/kiam-helm-charts/charts/
$ helm repo update
$ helm install uswitch/kiam
```

More detailed Helm usage can be found on the Helm Hub:

[Kiam - Helm Hub](https://hub.helm.sh/charts/uswitch/kiam)

## How it Works
Kiam is split into two processes that run independently.

### Agent
This is the process that would typically be deployed as a DaemonSet to ensure that Pods have no access to the AWS Metadata API. Instead, the agent runs an HTTP proxy which intercepts credentials requests and passes on anything else. An DNAT iptables [rule](cmd/kiam/iptables.go) is required to intercept the traffic. The agent is capable of adding and removing the required rule for you through use of the `--iptables` [flag](cmd/kiam/agent.go). This is the name of the interface where pod traffic originates and it is different for the various CNI implementations. The flag also supports the `!` prefix for inverted matches should you need to match all but one interface.

##### Typical CNI Interface Names #####

| CNI | Interface | Notes |
|-----|-----------|-------|
| [amazon-vpc-cni-k8s](https://github.com/aws/amazon-vpc-cni-k8s) and [cni-ipvlan-vpc-k8s](https://github.com/lyft/cni-ipvlan-vpc-k8s) | `!eth0` | This CNI plugin attaches multiple ENIs to the instance. Typically eth1-ethN (N depends on the instance type) are used for pods which leaves eth0 for the kubernetes control plane. The ! prefix on the interface name inverts the match so metadata service traffic from all interfaces except eth0 will be sent to the kiam agent. *Requires kiam v2.7 or newer.* |
| [weave](https://www.weave.works/docs/net/latest/kubernetes/kube-addon/) | `weave` |   |
| [calico/canal](https://docs.projectcalico.org/v3.1/getting-started/kubernetes/installation/flannel) | `cali+` |   |
| [kube-router](https://www.kube-router.io/docs) | `kube-bridge` | This is the default bridge interface that all the pods are connected to when using kube-router |
| [kops (kubenet)]( https://github.com/kubernetes/kops/blob/master/docs/networking.md) | `cbr0` | When using the default CNI (kubenet) |
| [cilium](https://docs.cilium.io/) | `lxc+` |  |


### Server
This process is responsible for connecting to the Kubernetes API Servers to watch Pods and communicating with AWS STS to request credentials. It also maintains a cache of credentials for roles currently in use by running pods- ensuring that credentials are refreshed every few minutes and stored in advance of Pods needing them.

## Building locally
If you want to build and run locally:
- `go version` >= 1.9
- run the following
```
$ mkdir -p $GOPATH/src/github.com/uswitch
$ git clone git@github.com:uswitch/kiam.git $GOPATH/src/github.com/uswitch/kiam
$ cd $GOPATH/src/github.com/uswitch/kiam
$ make
```

## License

```
Copyright 2017 uSwitch

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## Thanks to Kube2iam
We owe a **huge** thanks to the creators and maintainers of [Kube2iam](https://github.com/jtblin/kube2iam) which we ran for many months as we were bootstrapping our clusters.

We wanted to overcome two things in kube2iam:

1. We had data races under load causing incorrect credentials to be issued [#46](https://github.com/jtblin/kube2iam/issues/46).
1. Prefetch credentials to reduce start latency and improve reliability.

Other improvements/changes we made were (largely driven out of how we have our systems setup):

1. Use structured logging to improve the integration into our ELK setup with pod names, roles, access key ids etc.
1. Use metrics to track response times, cache hit rates etc.
