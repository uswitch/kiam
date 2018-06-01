# Changelog

## v2.8
1st June 2018

Notable changes:

* [#62](https://github.com/uswitch/kiam/pull/62) Documented interfaces to specify when using Kiam with amazon-vpc-cni.
* [#76](https://github.com/uswitch/kiam/pull/76) Wait for balancer to have addresses in Gateway. This removes a bunch of misleading error messages from the healthcheck command.

Thanks to the following people for contributing in this release:

* [sp-joseluis-ledesma](https://github.com/sp-joseluis-ledesma)
* [ripter](https://github.com/ripta)

## v2.7
30th April 2018

Notable changes:

* Fix [Issue 43](https://github.com/uswitch/kiam/issues/43): updates to metadata api paths on m5/c5 instances
* [#41](https://github.com/uswitch/kiam/pull/41): Server allows for custom STS session durations with `--session-duration`
* Server uses `cache.NewIndexerInformer` to maintain pod and namespace caches, this also addresses an error identified in [Issue 46](https://github.com/uswitch/kiam/issues/46). 
* [#54](https://github.com/uswitch/kiam/pull/54) Agents can use a `!` prefix on interfaces when configuring iptables rules. This makes it possible to use Kiam with Amazon and Lyft's CNI plugins.
* Servers will wait for the pod and namespache caches to perform a sync with the Kubernetes API server before accepting gRPC connections. This may cause servers to take longer to start but ensures they have recent state before performing any operations.

Thanks to the following additional people for contributing/helping in this release:

* [elafarge](https://github.com/elafarge)
* [sami9gag](https://github.com/sami9gag)
* [mikesplain](https://github.com/mikesplain)
* [polarbizzle](https://github.com/polarbizzle)
* [Joseph-Irving](https://github.com/Joseph-Irving)
