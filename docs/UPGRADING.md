# Upgrading

## v2 to v3

Kiam changed significantly between v2.X and v3.0. Breaking changes are:

* The gRPC API was changed. v3 Agent processes can only connect and communicate with v3 Server processes.
* The Agent metadata proxy HTTP server now blocks access to any path other than those used for obtaining credentials.
* Server's handling of TLS has changed to remove port from Host. This requires certificates to name `kiam-server` rather than `kiam-server:443`, for example. Any issued certificates will likely need re-issuing.
* Separated agent, server and health commands have been merged into a kiam binary. This means that when upgrading the image referenced the command and arguments used will also need to change.

We would suggest upgrading in the following way:

1. Generate new TLS assets. You can use [docs/TLS.md](docs/TLS.md) to create new certificates, or use something like [cert-manager](https://github.com/jetstack/cert-manager) or [Vault](https://vaultproject.io). Given the TLS changes make sure that your server certificate supports names:
    * `kiam-server`
    * `kiam-server:443`
    * `127.0.0.1`
2. Create a new DaemonSet to deploy the v3 Server processes and should use the new TLS assets deployed above. This will ensure that you have new server processes running alongside the old servers. Once the v3 servers are running and passing their health checks you can proceed.
3. Update the Agent DaemonSet to use the v3 image. Because the command has changed it's worth being careful when changing this as the existing configuration will not work with v3. One option is to ensure your DaemonSet uses a `OnDelete` [update strategy](https://kubernetes.io/docs/tasks/manage-daemon/update-daemon-set/#daemonset-update-strategy): you can deploy new nodes running new agents connecting to new servers while leaving existing nodes as-is. 