# Helm Chart Changelog

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
* [#303](https://github.com/uswitch/kiam/pull/303) Added support for imagePullSecrets

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
