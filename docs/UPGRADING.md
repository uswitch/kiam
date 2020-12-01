# Upgrading

## v3 to v4

Kiam changed significantly between v3.X and v4.0. Breaking changes are:

- The role policy is now applied after the role ARN has been resolved, this may cause compatibility issues with existing `iam.amazonaws.com/permitted` restrictions.
- StatsD metrics have been removed.
- The agent gRPC keepalve flags have been renamed.

When upgrading you will want to ensure that you check the following:

1. Ensure your `iam.amazonaws.com/permitted` annotations take into account that the regex will now be evaluated on the resolved role ARN, it is now possible that v3.X rules become more permissive in some scenarios, and less permissive in others.
    * Given you previously had a restriction like `iam.amazonaws.com/permitted=^test-role$` and a Pod using the role `iam.amazonaws.com/role=test-role` the role would now not be permitted as the regex would not match when evaluated against the full role ARN `arn:aws:iam::1234567890:role/test-role`.
    * Given you previously had a restriction like `iam.amazonaws.com/permitted=.*test-role` and a Pod using the role `arn:aws:iam::1234567890:role/test-role` the role would now be permitted as the regex matches when evaluated against the full role ARN.
2. If you still require StatsD metrics you may need to look at something like [veneur-prometheus](https://github.com/stripe/veneur/tree/master/cmd/veneur-prometheus) to scrape the /metrics endpoint and push them to StatsD.
3. Ensure you use the new gRPC keepalive flags when configuring the agent.
    * `--grpc-keepalive-time-ms` becomes `-grpc-keepalive-time-duration`
    * `--grpc-keepalive-timeout-ms` becomes `--grpc-keepalive-timeout-duration`

### Helm

If you are using Helm to install Kiam, be sure to use the latest 4.x chart when upgrading.

## v2 to v3

Kiam changed significantly between v2.X and v3.0. Breaking changes are:

* The gRPC API was changed. v3 Agent processes can only connect and communicate with v3 Server processes.
* The Agent metadata proxy HTTP server now blocks access to any path other than those used for obtaining credentials.
* Server's handling of TLS has changed to remove port from Host. This requires certificates to name `kiam-server` rather than `kiam-server:443`, for example. Any issued certificates will likely need re-issuing.
* Separated agent, server and health commands have been merged into a kiam binary. This means that when upgrading the image referenced the command and arguments used will also need to change.
* Server now reports events to Pods, requiring additional RBAC privileges for the service account.

We would suggest upgrading in the following way:

1. Generate new TLS assets. You can use [docs/TLS.md](TLS.md) to create new certificates, or use something like [cert-manager](https://github.com/jetstack/cert-manager) or [Vault](https://vaultproject.io). Given the TLS changes make sure that your server certificate supports names:
    * `kiam-server`
    * `kiam-server:443`
    * `127.0.0.1`
2. Create a new DaemonSet to deploy the v3 Server processes and should use the new TLS assets deployed above. This will ensure that you have new server processes running alongside the old servers. Once the v3 servers are running and passing their health checks you can proceed. **Please note that RBAC policy changes are required for the Server** and are documented in [deploy/server-rbac.yaml](../deploy/server-rbac.yaml)
3. Update the Agent DaemonSet to use the v3 image. Because the command has changed it's worth being careful when changing this as the existing configuration will not work with v3. One option is to ensure your DaemonSet uses a `OnDelete` [update strategy](https://kubernetes.io/docs/tasks/manage-daemon/update-daemon-set/#daemonset-update-strategy): you can deploy new nodes running new agents connecting to new servers while leaving existing nodes as-is.
