# kiam
kiam runs as an agent on each node in your Kubernetes cluster and allows cluster users to associate IAM roles to Pods.

Docker images are available at [https://quay.io/repository/uswitch/kiam](https://quay.io/repository/uswitch/kiam).

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

Further, all namespaces must also have an annotation with a regular expression expressing which roles are permitted to be assumed within that namespace. Without the namespace annotation the pod will be unable to assume any roles. 

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

## How it Works
Kiam is split into two processes that run independently.

### Agent
This is the process that would typically be deployed as a DaemonSet to ensure that Pods have no access to the AWS Metadata API. Instead, the agent runs an HTTP proxy which intercepts credentials requests and passes on anything else. 

### Server
This process is responsible for connecting to the Kubernetes API Servers to watch Pods and communicating with AWS STS to request credentials. It also maintains a cache of credentials for roles currently in use by running pods- ensuring that credentials are refreshed every few minutes and stored in advance of Pods needing them.

## Building locally
If you want to build and run locally you can

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

