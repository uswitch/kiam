# Changelog

## v3.0
XX December 2018

v3 introduces a change to the gRPC API. Servers are compatible with v2.x Agents, although v3 Agents require v3 Servers. A few other changes mean it's worth reading through [docs/UPGRADING.md](docs/UPGRADING.md) for more detail on moving from v2 to v3.

Notable changes:

* [#109](https://github.com/uswitch/kiam/pull/109) v3 API
* [#110](https://github.com/uswitch/kiam/pull/110) Restrict metadata routes. Everything other than credentials **will be blocked by default**
* [#122](https://github.com/uswitch/kiam/pull/122) Record Server error messages as Events on Pod
* [#131](https://github.com/uswitch/kiam/pull/131) Replace go-metrics with native Prometheus metrics client
* [#140](https://github.com/uswitch/kiam/pull/140) Example Grafana dashboard for Prometheus metrics
* [#163](https://github.com/uswitch/kiam/pull/163) Server manifests use 127.0.0.1 rather than localhost to avoid DNS
* [#173](https://github.com/uswitch/kiam/pull/173) Metadata Agent uses 301 rather than 308 redirects
* [#180](https://github.com/uswitch/kiam/pull/180) Fix race condition with xtables.lock
* [#193](https://github.com/uswitch/kiam/pull/193) Add optional pprof http handler to add monitoring in live clusters

A huge thanks to the following contributors for this release:

* [@Joseph-Irving](https://github.com/Joseph-Irving)
* [@max-lobur](https://github.com/max-lobur)
* [@fernandocarletti](https://github.com/fernandocarletti)
* [@integrii](https://github.com/integrii)
* [@duncward](https://github.com/duncward)
* [@stevenjm](https://github.com/stevenjm)
* [@tasdikrahman](https://github.com/tasdikrahman)
* [@word](https://github.com/word)
* [@DewaldV](https://github.com/DewaldV)
* [@roffe](https://github.com/roffe)
* [@sambooo](https://github.com/sambooo)
* [@idiamond-stripe](https://github.com/idiamond-stripe)
* [@ash2k](https://github.com/ash2k)
* [@moofish32](https://github.com/moofish32)
* [@sp-joseluis-ledesma](https://github.com/sp-joseluis-ledesma)

## v2.8
1st June 2018

Notable changes:

* [#62](https://github.com/uswitch/kiam/pull/62) Documented interfaces to specify when using Kiam with amazon-vpc-cni.
* [#76](https://github.com/uswitch/kiam/pull/76) Wait for balancer to have addresses in Gateway. This helps prevent the following errors being reported by the health check command:
```
WARN[0000] error checking health: rpc error: code = Unavailable desc = there is no address available 
```

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
