[![Go Documentation](https://godocs.io/github.com/richardjennings/simple-ops?status.svg)](https://godocs.io/github.com/richardjennings/simple-ops)
[![codecov](https://codecov.io/gh/richardjennings/simple-ops/branch/main/graph/badge.svg?token=TLYP6632YV)](https://codecov.io/gh/richardjennings/simple-ops)
![example branch parameter](https://github.com/richardjennings/simple-ops/actions/workflows/test-coverage.yml/badge.svg?branch=main)
[![DeepSource](https://deepsource.io/gh/richardjennings/simple-ops.svg/?label=active+issues&show_trend=true&token=3QKEWbwDSIK8u6X0iXP3Spuo)](https://deepsource.io/gh/richardjennings/simple-ops/?ref=repository-badge)
[![DeepSource](https://deepsource.io/gh/richardjennings/simple-ops.svg/?label=resolved+issues&show_trend=true&token=3QKEWbwDSIK8u6X0iXP3Spuo)](https://deepsource.io/gh/richardjennings/simple-ops/?ref=repository-badge)

# Simple-Ops

Simple-Ops is a GitOps repository management tool.

## Why

Simple-Ops promotes repeatability and consistency as first class features with the repository acting as a single source of truth.
The [Verify command](#Verify) rebuilds all deployment manifests and compares the result with the current deployment manifests.
This provides a convenient mechanism as a CI check or pre-receive hook that gives confidence in the correctness of manifests.

Simple-Ops promotes the use of charts vendored in the repository as tgz files and does not support fetching remote charts at 
run-time. This 'vendoring' further improves reliability and resilience aiming to make the [Generate command](#Generate) a pure function.

Simple-Ops leverages a composition pattern to make decorating charts with ancillary K8s manifests such as
SealedSecrets config driven using templating. Optionally decorating manifests can be written to a path outside ```./deploy/``` 
for example ```./apps/``` for Argo-CD Applications.

Simple-Ops wraps functionality from Helm v3, Kustomize and Jsonnet providing an opinionated workflow whilst remaining compatible with
other tools.

## Get Started
Binaries for Linux and Mac for both AMD64 and ARM64 are available via the [Releases Page](https://github.com/richardjennings/simple-ops/releases)

Multi-arch (amd64, arm64) container images are available at [https://hub.docker.com/repository/docker/richardjennings/simple-ops](https://hub.docker.com/repository/docker/richardjennings/simple-ops)

For an example use of Simple-Ops to manage a GitOps repository see [Simple Ops Example](https://github.com/richardjennings/simple-ops-example)

Simple-Ops can be used via a GitHub Actions implementation at [https://github.com/richardjennings/simple-ops-action](https://github.com/richardjennings/simple-ops-action)

## Usage

### Add
Add a chart locally, for example ```simple-ops add sealed-secrets --repo https://bitnami-labs.github.io/sealed-secrets --version 2.1.8```.
This results in a file at ```chart/sealed-secrets-2.1.8.tgz``` Optionally provide --add-config to generate a config file named
after the component, e.g. ```config/sealed-secrets.yml``` with content ```chart: sealed-secrets-2.1.8.tgz```.
The chart version, name, repository and a sha256 digest of the .tgz content are recorded in the simple-ops.lock file.


### Container-Resources
Lists all Resource configurations for Container specs in generated manifests either globally or per deployment.

### Generate
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
Show is a wrapper around ```helm show``` based on Simple-Ops deployments. For example: ```simple-ops show values production.myapp```
would show the helm chart values associated with the production environment myapp component chart.

### Verify
Verify runs Generate but does not update the deployment directory with any changes. It performs a comparison using
SHA256 and reports if the `/tmp/deploy` directory content matches ```/my/project/deploy``` content.
Verify also checks that all tgz charts in the charts directory are represented in the simple-ops.lock file and that the
sha256 hash of each chart.tgz matches that recorded in the lock file.

#### Note
Some charts may dynamically generate random data in rendered chart templates. For example Redis creates a random password
secret by default. This will result in ```simple-ops verify``` failing as the generated output does not exactly match the
previously generated output. The resolution is to ensure chart templates are idempotent by handling such cases explicitly.



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
  name: sealed-secrets 
deploy:
  local:
    namespace:
      name: kube-system
  staging:
    chart: sealed-secrets-2.1.7.tgz
    namespace:
      create: true
```

will result in the following deploy configurations:
```yaml
# local.sealed-secrets
chart: sealed-secrets-2.1.8.tgz
namespace:
  name: kube-system
  create: false
  inject: true
```

```yaml
# staging.sealed-secrets
chart: sealed-secrets-2.1.7.tgz
namespace:
  name: sealed-secrets
  create: true
  inject: true
```


The components of configuration are:
```yaml
chart: <string> # filename or directory name in charts/
namespace: <map>
   name: <string> # name of namespace
   create: <bool> # generate a namespace manifest or not
   injecct: <bool> # inject namespace config into resources (after helm templating)
labels: <map>
   key: value # label name to label value map
disabled: <bool> # disable the configuration
with: <map> # ad-hoc templates
   template: <map> # the name sans .yml in /resources/
      path: <string> # optionally render manifest to file relative to project e.g. ./apps/myapp.yaml
      values: <map> # values merged into with template yaml configuration
         example: value
values: <map> values to pass to Helm templating
fsslice: <map> configuration of kustomize filterspec's
deploy: <map> # deploy specifies the per environment configuration for a component
   environment-name: <config> # the configuration is identical to the parent sans deploy
kustomizations: #<map> of name to Kustomization yaml
jsonnet: #<map> of name to Jsonnet configuration
  name:
    path: <string> # path to jsonnet file
    values: <map> # key values pairs as Jsonnet external variables
    inline: <string> # Jsonnet program declared inline
preservePaths: #<list> string any relative directory paths required by the generate stage (copied to tmp build context)
```

The global config ```simple-ops.yml``` is merged with the component config. Any defaults specified globally can
be overriden on a component level.
Deploy configurations are pulled from the component configuration and have the component
configuration merged into them.

## With
With components are yaml manifests. A with component can have values changed when used in a deploy config. For example:
```yaml
# application.yml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  destination:
    server: https://kubernetes.default.svc
  project: default
  source:
    directory:
      recurse: true
    repoURL: ssh://git@github.com/richardjennings/simple-ops-example.git
    targetRevision: HEAD
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

```yaml
# config/crossplane.yaml
deploy:
   example:
      with:
         application:
            crossplane:
               path: apps/example/crossplane.yaml
               values:
                  spec:
                     source:
                        path: deploy/example/crossplane/
```

The deployment generates a ```kind: Application``` manifest at ./apps/example/crossplane.yaml where spec.source.path
is changes (added) to ```deploy/example/crossplane/```

If path is not specified the generated With manifest is bundled with any helm generated manifests into 
```deploy/environment/component/manifest.yaml```. For example:

```yaml
# resources/sealed-secret.yml
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
spec:
```

```yaml
# deploy/argo-cd.yml
chart: argo-cd-4.6.2.tgz
deploy:
   example:
      values:
      with:
        sealed-secrets:
          argocd-repo-github:
            values:
               spec:
                  encryptedData:
                     name: AgBifiAijX0iZMK...
                     url: AgAEW+jbQNenKpqo...
                  template:
                     metadata:
                        labels:
                           argocd.argoproj.io/secret-type: repository
```
creates a sealed secrets manifest called argocd-repo-github appended to ```./deploy/example/argo-cd/manifest.yaml```

## Alternatives
### Argo-CD
### Cue
### Flux 2
### Helm
### Helmfile
### Helmwave
### Jsonnet
### Kaptain
### Kustomize
### Pulumi
### Tanka
### Kubecfg

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

