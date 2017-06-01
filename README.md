# kiam
kiam runs as an agent on each node in your Kubernetes cluster and allows cluster users to associate IAM roles to Pods.

[![Docker Pulls](https://img.shields.io/docker/pulls/uswitch/kiam.svg)]()
[![CircleCI](https://img.shields.io/circleci/project/github/uswitch/kiam.svg)]()

## Overview
From the [AWS documentation on IAM roles](http://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html):

> [a] role is similar to a user, in that it is an AWS identity with permission policies that determine what the identity can and cannot do in AWS. However, instead of being uniquely associated with one person, a role is intended to be assumable by anyone who needs it. Also, a role does not have any credentials (password or access keys) associated with it. Instead, if a user is assigned to a role, access keys are created dynamically and provided to the user.

kiam uses an annotation added to a `Pod` to indicate which role should be assumed. For example:

```yaml
kind: Pod
metadata:
  name: foo
  annotations:
    iam.amazonaws.com/role: reportingdb-reader
```

When your process starts an AWS SDK library will normally use a chain of credential providers (environment variables, instance metadata, config files etc.) to determine which credentials to use. kiam intercepts the metadata requests and uses the [Security Token Service](http://docs.aws.amazon.com/STS/latest/APIReference/Welcome.html) to retrieve temporary role credentials. 

## Deploying to Kubernetes

Please see [`./kiam.daemonset.yaml`](kiam.daemonset.yaml) for an example of how to deploy as a `DaemonSet` on Kubernetes.

Images are automatically pushed to Docker Hub: [uswitch/kiam](https://hub.docker.com/r/uswitch/kiam). Image releases are tagged with `latest` and their corresponding git tag `v1.0.1`. Master is also built and tagged as `latest-head` and the git sha.

## Usage
```
$ kiam --help
usage: kiam --role-base-arn=ROLE-BASE-ARN --host=HOST [<flags>]

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
      --json-log               Output log in JSON
  -d, --debug                  Log at Debug level
      --kubeconfig=KUBECONFIG  Path to kube config
      --port=3100              HTTP port
      --sync-interval=2m       Interval to refresh pod state from API server
      --allow-ip-query         Allow client IP to be specified with ?ip. Development use only.
      --role-base-arn=ROLE-BASE-ARN  
                               Base ARN for roles. e.g. arn:aws:iam::123456789:role/
      --statsd=""              UDP address to publish StatsD metrics. e.g. 127.0.0.1:8125
      --statsd-interval=10s    Interval to publish to StatsD
      --iptables               Add IPTables rules
      --host=HOST              Host IP address.
      --host-interface="docker0"  
                               Network interface for pods to configure IPTables.
```

## How it Works
kiam is split into a few processes:

* Web server. This handles requests from Pods when determining which role they should assume (using the Kubernetes cache) and retrieving credentials for the role (using the Credentials cache).
* Kubernetes cache. This uses a Watch to monitor changes on the cluster and runs a periodic sync (via. a List) to ensure a local cache of Pods.
* Credentials cache. This uses the AWS API to retrieve session credentials and stores them in an in-memory cache.
* Prefetch. This watches for changes in the Kubernetes cache and warms the Credentials cache for uncompleted Pods. It is also notified when Credentials expire from the credentials cache and determines whether they should be refreshed or discarded.

It is currently intended to be run as a `DaemonSet`- running a kiam process on each node in your cluster.

## Thanks to Kube2iam
We owe a **huge** thanks to the creators and maintainers of [Kube2iam](https://github.com/jtblin/kube2iam) which we ran for many months as we were bootstrapping our clusters.

We wanted to overcome two things in kube2iam:

1. We had data races under load causing incorrect credentials to be issued [#46](https://github.com/jtblin/kube2iam/issues/46).
1. Prefetch credentials to reduce start latency and improve reliability.

Other improvements/changes we made were (largely driven out of how we have our systems setup):

1. Use structured logging to improve the integration into our ELK setup with pod names, roles, access key ids etc.
1. Use metrics to track response times, cache hit rates etc.

## Building locally
If you want to build and run locally you can

```
$ mkdir -p $GOPATH/src/github.com/uswitch
$ git clone git@github.com:uswitch/kiam.git $GOPATH/src/github.com/uswitch/kiam
$ cd $GOPATH/src/github.com/uswitch/kiam
$ go build -o bin/kiam cmd/*.go
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