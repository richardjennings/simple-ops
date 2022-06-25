[![Go Documentation](https://godocs.io/github.com/richardjennings/simple-ops?status.svg)](https://godocs.io/github.com/richardjennings/simple-ops)
[![codecov](https://codecov.io/gh/richardjennings/simple-ops/branch/main/graph/badge.svg?token=TLYP6632YV)](https://codecov.io/gh/richardjennings/simple-ops)
![example branch parameter](https://github.com/richardjennings/simple-ops/actions/workflows/test-coverage.yml/badge.svg?branch=main)

# Simple-Ops

Simple-Ops is a GitOps repository management tool.

## Why
There is a lack of tooling available specifically designed to make managing a GitOps repository simple.

Simple-Ops promotes repeatability and consistency as first class features. The [Verify command](#Verify) rebuilds all deployment
manifests and compares the result with the current deployment manifests. This provides a convenient mechanism as a CI check or
pre-receive hook that gives confidence in the correctness of manifests.

Simple-Ops promotes the use of charts vendored in the repository as tgz files and does not support fetching remote charts at 
run-time. This 'vendoring' further improves reliability and resilience aiming to make the [Generate command](#Generate) a pure function.

Simple-Ops leverages a composition pattern to make decorating charts with ancillaries such as Argo-CD Application manifests,
SealedSecrets, Istio configuration, and any other K8s manifests configuration driven using templating and optionally bespoke
directory paths for manifest output.

Simple-Ops wraps functionality from Helm v3 and Kustomize providing an opinionated workflow whilst remaining compatible with
other tools.

## Get Started
Binaries for Linux and Mac for both AMD64 and ARM64 are available via the [Releases Page](https://github.com/richardjennings/simple-ops/releases)

Multi-arch (amd64, arm64) container images are available at [https://hub.docker.com/repository/docker/richardjennings/simple-ops](https://hub.docker.com/repository/docker/richardjennings/simple-ops)

For an example use of Simple-Ops to manage a GitOps repository see [Simple Ops Example](https://github.com/richardjennings/simple-ops-example)

Simple-Ops can be used via a GitHub Actions implementation at [https://github.com/richardjennings/simple-ops-action](https://github.com/richardjennings/simple-ops-action)

## Configuration

Configuration is available globally via simple-ops.yml, on a component basis by top level keys in config/component.yml
and per environment by configuration keys within environment configuration. For example:
```yaml
# simple-ops.yml
namespace:
  name: default
  create: false
  inject: true
```
```yaml
# config/sealed-secrets.yml
chart: sealed-secrets-2.1.8.tgz
namespace:
  name:   sealed-secrets 
  create: true
  inject: true
deploy:
  local:
    namespace:
      name: kube-system
  staging:
    namespace:
      name: sealed-secrets
      create: true
```
renders the sealed-secrets-2.1.8.tgz chart to both deploy/local/sealed-secrets/manifests.yaml and deploy/staging/sealed-secrets/manifests.yaml, 
both with a namespace resource defined with name kube-system and sealed-secrets respectively, and the namespace configuration
injected into all relevant resources via a kustomize filter.

## Usage

### Add
Add a chart locally, for example ```simple-ops add sealed-secrets --repo https://bitnami-labs.github.io/sealed-secrets --version 2.1.8```.
This results in a file at ```chart/sealed-secrets-2.1.8.tgz``` Optionally provide --add-config to generate a config file named
after the component, e.g. ```config/sealed-secrets.yml``` with content ```chart: sealed-secrets-2.1.8.tgz```


### Container-Resources
Lists all Resource configurations for Container specs in generated manifests either globally or per deployment.

### <a id="Verify"></a> Generate
Renders all Helm charts configured to corresponding deployment directories. 
Performs labelling and namespace customisations and generates all templated 'with' ancillaries.

### Images
Lists all images either globally or per deployment

### Init
Creates the default Simple-Ops directory structure and generates a default ```simple-ops.yml```

### Set
Add a configuration option to a deployment. For example ```simple-ops set myapp.deploys.staging.values.imgSrc my-container:${SHA}```
would add or update the imgSrc value passed to Helm rendering to some value. This process can be used to allow multiple
components in a deployment pipeline to construct a unified deployment PR.

### Show
Show is wrapper around ```helm show``` based on Simple-Ops deployments. For example: ```simple-ops show values production.myapp```
would show the helm chart values associated with the production environment myapp component chart.

### <a id="Verify"></a> Verify
Verify runs Generate but does not update the deployment directory with any changes. It performs a comparison using
SHA256 and reports if the `/tmp/deploy` directory content matches ```/my/project/deploy``` content.

### Note
Some charts may dynamically generate random data in rendered chart templates. For example Redis creates a random password
secret by default. This will result in ```simple-ops verify``` failing as the generated output does not exactly match the
previously generated output. The resolution is to ensure chart templates are idempotent by handling such cases explicitly.


## Key tenants
* Repeatability for verification and consistency.
* Local and complete dependencies for reliability and resilience.
* Composition for extensibility.
* Narrow scope for compatability with a large range of usage patterns.




## GitOps Principles
[Weaveworks Vendor Neutral GitOps](https://www.weave.works/blog/opengitops-the-vendor-neutral-gitops-project)

1. A desired state as expressed in a declarative system
    - Simple-Ops expects dependencies to be vendored explicitly, subscribing to the 'single source of truth' philosophy
2. Immutable versions of that desired state
    - Everything gets recorded: configuration changes are managed via machine as simple-ops set or by hand and are 
   intended to be committed to Git with ```simple-ops verify``` passing.
3. Continuous state reconciliation
    - Not covered by Simple-Ops
4. Operations through declaration
    - The generated and verified configuration managed by Simple-Ops including the configuration to verify and regenerate
   deployment manifests updated either by CI/CD or by hand are made available via Git repository. 

