# Helm Chart Changelog
# 6.1.0
17 May 2021

Notable changes:
* [#474](https://github.com/uswitch/kiam/pull/474) Add option to disable strict namespace regexp mode

Many thanks to the following contributor for this release:
* [@wcarlsen](https://github.com/wcarlsen)

# 6.0.0
4 January 2021

Notable changes:
* [#439](https://github.com/uswitch/kiam/pull/439) Kiam v4 release compatibility

Many thanks to the following contributor for this release:
* [@stefansedich](https://github.com/stefansedich)

# 5.10.0
18 August 2020

Notable changes:
* [#415](https://github.com/uswitch/kiam/pull/415) Allow disabling of mounting SSL certs from host

Many thanks to the following contributor for this release:
* [@velothump](https://github.com/velothump)

## 5.9.0
17 August 2020

Notable changes:
* [#417](https://github.com/uswitch/kiam/pull/417) Update kiam release from 3.5 to 3.6.

Many thanks to the following contributor for this release:
* [@leosunmo](https://github.com/leosunmo)

## 5.8.1
12 June 2020

Notable changes:
* [#406](https://github.com/uswitch/kiam/pull/406) Fix indentation on seLinuxOptions in server daemonset in Helm chart.

Many thanks to the following contributor for this release:
* [@mbarrien](https://github.com/mbarrien)


## 5.8.0
11 June 2020

Notable changes:
* [#404](https://github.com/uswitch/kiam/pull/404) Allow for configurable SELinux labels. Adds seLinuxOptions values to the Helm chart for both the agent and server.

Many thanks to the following contributor for this release:
* [@hfuss](https://github.com/hfuss)

## 5.7.1
11 June 2020

Notable changes:
* [#405](https://github.com/uswitch/kiam/pull/405) Fix incorrect indentation in server's pod affinity.

Many thanks to the following contributor for this release:
* [@msvechla](https://github.com/msvechla)

## 5.7.0
5 February 2020

Notable changes:
* [#367](https://github.com/uswitch/kiam/pull/367) Add possibility to configure agent/server initContainers from the `values.yaml`.

Many thanks to the following contributor for this release:
* [@caiohasouza](https://github.com/caiohasouza)

## 5.6.1
4 February 2020

Notable changes:
* [#372](https://github.com/uswitch/kiam/pull/372) Add missing sslCertHostPath as allowedHostPath in Helm Chart server PSP. Also fixes misplaced `extraHostPathMounts` range in the PSP.

Many thanks to the following contributor for this release:
* [@phyrog](https://github.com/phyrog)

## 5.6.0
24 January 2020

Notable changes:
* [#361](https://github.com/uswitch/kiam/pull/361) Add ability to configure the agent and server readiness/liveness probes from the `values.yaml`.

Many thanks to the following contributor for this release:
* [@caiohasouza](https://github.com/caiohasouza)

## 5.5.0
13 January 2020

Notable changes:
* [#353](https://github.com/uswitch/kiam/pull/353) Update kiam release from 3.4 to 3.5.
* Optional gRPC keepalive [#337](https://github.com/uswitch/kiam/pull/337) configuration has been added to the chart for the agent under the `keepaliveParams:` field.

Many thanks to the following contributor for this release:
* [@johnmccabe](https://github.com/johnmccabe)

## 5.4.0
10 December 2019

Notable changes:
* [#336](https://github.com/uswitch/kiam/pull/336) Add optional service-account annotations for server and agent. Fix deployment to work with helm v3 by removing the erroneous `updateStrategy` field and add support for the custom SSL host path.

Many thanks to the following contributor for this release:
* [@tyrken](https://github.com/tyrken)

## 5.3.0
27 November 2019

Notable changes:
* [#332](https://github.com/uswitch/kiam/pull/332) The option of running the Kiam server component as a Deployment, rather than a Daemonset, has been added to the chart - this can be configured in the `values.yaml`.

Many thanks to the following contributor for this release:
* [@denniswebb](https://github.com/denniswebb)

## 5.2.0
27 November 2019

Notable changes:
* [#320](https://github.com/uswitch/kiam/pull/320) The default SSL host path set for the Kiam server has been updated to match the default in the repo's deployment manifests. This path can now be configured from its own `values.yaml` option.  
Also, the Helm README has been updated to include documentation for key configuration elements.

Many thanks to the following contributors for this release:
* [@MVJosh](https://github.com/MVJosh)
* [@Nuru](https://github.com/Nuru)

## 5.1.0
8 November 2019

Notable changes:
* [#319](https://github.com/uswitch/kiam/pull/319) Optional [deep liveness check](https://github.com/uswitch/kiam/pull/268) has been added to the helm chart.

Many thanks to the following contributor for this release:
* [@stanvit](https://github.com/stanvit)

## 5.0.0
7 November 2019

**BREAKING CHANGES:**
* [#322](https://github.com/uswitch/kiam/pull/322) The chart has been updated to include support for the `--no-iptables-remove` Kiam flag, which is now **enabled by default.**  
`Note:` Using your existing `values.yaml` with this chart will result in the flag being turned on with no user input. This flag leaves the iptables rule (which is necessary for the Kiam agent processes to intercept requests to the metadata API) in place after the Kiam agent has been shutdown.  
Please see the following links for related discussion: [Issue 202](https://github.com/uswitch/kiam/issues/202) and [PR #253](https://github.com/uswitch/kiam/pull/253).

Many thanks to the following contributors for this release:
* [@zytek](https://github.com/zytek)
* [@Nuru](https://github.com/Nuru)

## 4.2.0
30 October 2019

Notable Changes:
* [#310](https://github.com/uswitch/kiam/pull/310) Updated the default `values.yaml` gatewayTimeoutCreation to 1s.

Many thanks to the following contributor for this release:
* [@zytek](https://github.com/zytek)

## 4.1.0
28 October 2019

Notable Changes:
* [#314](https://github.com/uswitch/kiam/pull/314) Added support for Prometheus Operator ServiceMonitors.

Many thanks to the following contributor for this release:
* [@mikhailadvani](https://github.com/mikhailadvani)

## 4.0.0
16 October 2019

**BREAKING CHANGES:**
* [#307](https://github.com/uswitch/kiam/pull/307) Upgraded Kubernetes Apps API version for the DaemonSets in order to support Kubernetes 1.16+.  
`Note:` This API change has the effect of dropping support for Kubernetes >1.9. This release WILL NOT work for Kubernetes clusters running versions earlier than 1.9.

Many thanks to the following contributor for this release:
* [@velothump](https://github.com/velothump)

## 3.2.0
10 October 2019

Notable Changes:
* [#303](https://github.com/uswitch/kiam/pull/303) Added support for imagePullSecrets.

Many thanks to the following contributor for this release:
* [@junaid18183](https://github.com/junaid18183)

## 3.1.0
07 October 2019

Notable Changes:
* [#304](https://github.com/uswitch/kiam/pull/304) Update kiam release from 3.3 to 3.4

Many thanks to the following contributor for this release:
* [@jeffb4](https://github.com/jeffb4)

## 3.0.1
10 September 2019

Notable Changes:
* [#295](https://github.com/uswitch/kiam/pull/295) **BUG FIX** - The Kiam server and agent daemonset files have been updated to account for the change to the `values.yaml` file made in [#292](https://github.com/uswitch/kiam/pull/292). Without this change, users will experience issues when deploying the v3 release of the Chart with the extraEnv parameters set in `values.yaml`.

## 3.0.0
5 September 2019

**BREAKING CHANGES:**
* [#292](https://github.com/uswitch/kiam/pull/292) The `extraEnv` parameters for both the agent and server in `values.yaml` have been changed to support an array of options. This adds support for creating env vars from configMaps or secretKeyRefs.  
`Note:` This will break any existing Helm deployments which utilise the `extraEnv` parameters in `values.yaml`. You will need to update your `values.yaml` file to match the format in the [template](/helm/kiam/values.yaml#L93)

## 2.5.3
29 August 2019

Notable Changes:
* [#288](https://github.com/uswitch/kiam/pull/288) Bug fix for correctly rendering port number for certificates.

Many thanks to the following contributor for this release:
* [@simnalamburt](https://github.com/simnalamburt)

## 2.5.2
27 August 2019

Notable Changes:
* [#285](https://github.com/uswitch/kiam/pull/285) and [#287](https://github.com/uswitch/kiam/pull/287) Chart is updated to include uSwitch logo.

## 2.5.1
20 August 2019

Notable Changes:
* [#283](https://github.com/uswitch/kiam/pull/283) Kiam Helm charts are added to the [uswitch/kiam](https://github.com/uswitch/kiam) repo.
