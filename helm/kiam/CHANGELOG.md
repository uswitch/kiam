# Helm Chart Changelog
## 3.0.1
10 September 2019

Notable Changes:
* [#295](https://github.com/uswitch/kiam/pull/295) <b>BUG FIX</b> - The Kiam server and agent daemonset files have been updated to account for the change to the `values.yaml` file made in [#292](https://github.com/uswitch/kiam/pull/292). Without this change, users will experience issues when deploying the v3 release of the Chart with the extraEnv parameters set in `values.yaml`.

## 3.0.0
5 September 2019

<b>BREAKING CHANGES</b>:
* [#292](https://github.com/uswitch/kiam/pull/292) The `extraEnv` parameters for both the agent and server in `values.yaml` have been changed to support an array of options. This adds support for creating env vars from configMaps or secretKeyRefs.</br>
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
