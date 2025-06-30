# Release Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/lunarway/release-manager)](https://goreportcard.com/report/github.com/lunarway/release-manager)

GitOps release manager for kubernetes configuration repositories.

This project is used as an internal project at Lunar and it therefore contains some assumptions on our setup. This includes environment naming (dev, prod). Further it is build around assumptions made by our OSS project `shuttle`, and id's for releases are a combination of branch name, git-sha from source repo, and git-sha from shuttle plan repo. Our initial intent is not to support this as an open source project.

We will however, have it public available for reference. This might change over time.

# Design

The release-manager consist of 4 different "microservices" with each having a specific responsibility in the pipeline. The applications are basically utilities for moving files around a Git repository.

A simplified overview of all the components involved in the flow can be seen below.

![](docs/gitops_workflow.png)

To read more about the what each service is responsible for see [Components](#components).
In addition to the services we utilize [Jenkins](https://github.com/jenkinsci/jenkins) as a CI server and [flux](https://github.com/fluxcd/flux) as a release operator running inside each cluster.

# Interactions

As seen on the illustration the main interaction point for developers are their service repository, doing code changes and with the `hamctl` CLI.

Below are descriptions of the common commands used in day to day activities.

## Promote

The promotion flows, is a convetion based release process. It can be invoked by `hamctl` as follows:

```
hamctl promote --service example --env dev
```

The convention follows the following flow: `master -> dev -> prod`
As seen in the example above, the `example` service will be promoted from the lastest available artifact from `master` to the `dev` environment.

Another example, is a promotion of an artifact running in, e.g. dev, to the production environment. This can be achieved with the following command:

```
hamctl promote --service example --env prod
```

The above locates what is running in the `dev` environment, and takes the necessary steps to run the same artifact in `prod`.

## Release

The release flow, is a more liberal release process. There is no conventions in how artifacts move between environments. This makes it suitable for releasing `hotfix`-branches to production or `feature`-branches to a specific environment for testing before merging into `master`.

The release flow currently consist of two approaches, either the release of the lastest artifact from a given branch, or a specific artifact id.

Example of a release of a feature branch to the `dev` environment:

```
hamctl release --service example --branch "feature/new_feature" --env dev
```

Example of a release of a specific artifact id to the `prod` environment:

```
hamctl release --service example --artifact main-0017d995e3-67e9d69164 --env prod
```

## Status

Status is a convience flow to display currently released artifact to the three different environments; `dev`,`prod`.

```
$ hamctl status --service example

dev:
  Tag: master-1c1508405e-67e9d69164
  Author: Kasper Nissen
  Committer: Peter Petersen
  Message: empty-commit-to-test-flow
  Date: 2019-04-01 11:14:26 +0200 CEST
  Link: https://jenkins.example.lunar.app/job/github/job/example-service/job/master/132/display/redirect
  Vulnerabilities: 0 high, 0 medium, 0 low

prod:
  Tag: master-8fgh08405e-67e9d69164
  Author: John John
  Committer: Hans Hansen
  Message: some-commit
  Date: 2019-04-01 11:14:26 +0200 CEST
  Link: https://jenkins.example.lunar.app/job/github/job/example-service/job/master/132/display/redirect
  Vulnerabilities: 0 high, 0 medium, 0 low
```

## Policies

It is possible to configure policies for releases with `hamctl`'s `policy` command and globally with flags on the `server`.

You can `list`, `apply` and `delete` policies for a specific service like below.

```
hamctl policy --service <service> list
hamctl policy --service <service> apply <policy>
hamctl policy --service <service> delete <policy-id> [<policy-id>]
```

See below for details on how to apply specific policies.

Some policies cannot be applied simultaniously as they semantically does not support each other.
An example is an `auto-release` policy releasing a branch not compatible with a `branch-restriction` policy.
These cases are validated when applying either of them.

### Auto-release artifacts from branches to environments

An `auto-release` policy instructs the release manager to deploy new artifacts from a specific branch into an environment.

Multiple policies can be applied for the same branch to different environments, e.g. release `master` artifacts to `dev` and `prod`.

This is an example of applying an auto-release policy for the product service for the `master` branch and `dev` environment.

```
hamctl policy --service example apply auto-release --branch master --env dev
```

### Branch restriction on environments

A `branch-restriction` policy instructs the release manager to only allow artifacts from specific branches to be released to an environment.
The `--branch-regex` flag defines a regular expression that is matched against the branch name on every release.

As an example, the following command applies a branch-restriction policy for the `example` service that only allows the `master` branch to be released to the `prod` environment.

```
hamctl policy --service example apply branch-restriction --env prod --branch-regex '^master$'
```

Another example is to allow only `master` or `hostfix/*` branches in `prod` like this.

```
hamctl policy --service example apply branch-restriction --env prod --branch-regex '^(master|hotfix\/.+)$'
```

It is not possible to create an auto-release policy for a non-matching branch to an environment that is protected by a branch-restriction policy.

Be aware that the regular expression should be as strict as possible otherwise you might get unexpected results.
A branch regex like `master` will also allow branch names like `refactor-master-worker`, so make sure to mark the start `^` and end `$` of the string.

The `server` can also enforce branch restrictions on all managed services by setting the `policy-branch-restrictions` flag.
It takes a comma seprated list of `<environment>=<branchRegex>` values.

```
server start --policy-branch-restrictions 'production=^master$,dev=^development$'
```

They will be visible with `hamctl policy list` but cannot by removed with `hamctl`.
It is also not possible to overwrite them with custom policies, e.g. changing branch of a globally restricted environment.

# Releases and policies

Release files are structured as shown below.
In the root are folders for each environment, e.g. `dev`, `prod`.
These folders contain a `releases` directory with kubernetes resource definitions of each namespace and their running applications.
If an artifact contains a Flux `Kustomization` (`apiVersion: kustomize.toolkit.fluxcd.io/v1` and `kind: Kustomization`) custom resource the release manager moves it into the `clusters` directory tree.
This is tailored to support Flux2.

A `policies` directory holds all recorded release policies.
These are stored as JSON files for each service.

```
.
├── policies
│   └── <service>.json
├── <environments>
├── dev
│   └── releases
│       ├── <namespaces>
│       └── dev
│           └── <service>
│               ├── artifact.json
│               ├── 01-configmap.yaml
│               ├── 02-db-configmap.yaml
│               ├── 40-deployment.yaml
│               └── 50-service.yaml
└── clusters
    ├── <environments>
    └── dev
        ├── <namespaces>
        └── dev
            └── <service>.yaml
```

When running `kubectl apply` files are applied to the cluster alphabetically so the following convention should be used by configuration generators.

```
00 CRDs
01-09 configmaps
10-19 secrets
20-29 volumes
30-39 rbac
40-49 deployments/daemonsets
50-59 service
60-69 ingress
```

Resources starting with `00_` will skip resource validation in the Lunar shuttle plans.
`CustomResourceDefintions` requires custom schemas which are usually not available so they should always start with `00_`.

# Components

The release-manager consists of four applications.

| Application | Description                                                                                                                                                          |
| ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| artifact    | a simple tool for generating an artifact.json blob with information from the CI pipeline                                                                             |
| daemon      | a daemon reporting events about cluster component status back to the release-manager server                                                                          |
| hamctl      | a CLI client for interacting with the release-manager server                                                                                                         |
| server      | the API-server where clients (hamtcl) connects to, and daemon reports events to. It further implements different flows, e.g., promote a release, release an artifact |

## Artifact

`artifact` is used to generate, what we refer to as artifacts.
These are a `json` file containing relevant information from the CI flow.
The id's of the artifacts, are composed of `<branch-name>-<application-git-sha>-<shuttle-plan-git-sha>`.

We use [shuttle](https://github.com/lunarway/shuttle) plans to centralize our CI pipeline definitions, which is why we include the version of the plan in the artifact id.

Here is a redacted example of a generated artifact specification.

```json
{
  "id": "dev-0017d995e3-67e9d69164",
  "application": {
    "sha": "0017d995e32e3d1998395d971b969bcf682d2085",
    "message": "fix something",
    "name": "example-service",
    ...
    "url": "https://github.com/lunarway/example-service/commits/0017d995e32e3d1998395d971b969bcf682d2085",
    "provider": "GitHub"
  },
  "ci": {
    "jobUrl": "https://jenkins.example.lunar.app/job/github/job/example-service/job/dev/84/display/redirect",
    "start": "2019-03-29T13:47:15.259380775+01:00",
    "end": "2019-03-29T13:49:57.686299407+01:00"
  },
  "shuttle": {
    "plan": {
      "message": "Support-new-feature",
      "url": "git://git@github.com:lunarway/lw-shuttle-go-plan.git"
    }
  },
  "stages": [
    {
      "id": "build",
      "name": "Build",
      "data": {
        "dockerVersion": "18.09.3",
        "image": "quay.io/lunarway/example",
        "tag": "dev-0017d995e3-67e9d69164"
      }
    }
  ]
}
```

## hamctl

`hamctl` is a CLI for interacting with the release-manager server.
Examples of commands are `hamctl release` or `hamctl status`.

See [Interactions](#interactions) for more examples.

It uses a oauth2 authentication model for interacting with the server.
Specifically the Device Authorization flow.

This must be set up using the environment variables:

`HAMCTL_OAUTH_IDP_URL` pointing to your IdP where there must be an endpoint `{idp-url}/v1/token` for exchanging tokens.

`HAMCTL_OAUTH_CLIENT_ID` which is the oauth2 client id.

`hamctl` will automatically initiate a login if you do not have a valid token on your system.
You can opt out of this behaviour by setting the environement variable `HAMCTL_OAUTH_AUTO_LOGIN=false`.

### Completions

Shell completions are available with the command `completion`.
The following commands will add completions to the current shell in either bash or zsh.

```
source <(hamctl completion bash)
source <(hamctl completion zsh)
```

For a more detailed installation instruction see the help output.

```
hamctl completion --help
```

## daemon

The `daemon` is an agent running in each of the kubernetes clusters and reports state changes in the environment back to the release-manager.
`daemon` needs access to the kubernetes API server, and can be configured using a `ServiceAccount`.

`daemon` uses a token-based authentication model for interacting with the release-manager.
This token can be set using the command-line argument `--auth-token` or the ENV variable: `DAEMON_AUTH_TOKEN`

## Server

The server is responsible for all the operations related to releasing new versions.
`hamctl` and `artifact` communicates with it over HTTP to initiate releases, register new artifacts etc.

In its simplest form it is responsible for moving files around a Git repository based on the commands it receives, eg. release artifact.

### Notifications

When releasing applications the server will notify different upstream services along with outputting an identifiable log useful for log aggregation statistics.

```
info  command/start.go:145  Release [dev]: verification (master-e8da185c2c-06249f1a78) by Bjørn Sørensen, author Bjørn Sørensen
```

A Slack message is pushed to a `#releases-<env>` Slack channel.

![](docs/slack_release_message.png)

Grafana is annotated with release metadata and tag `deployment` useful for plotting on graphs to see when new releases are rolled out.

![](docs/grafana_annotation.png)

If the artifact provider is GitHub and a GitHub API token is provided (`--github-api-token`) the application source repository is tagged with `<env>` on the released Git SHA.

![](docs/github_tag.png)

### Tracing support

The server collects [Jaeger](https://www.jaegertracing.io/) spans. This is enabled by default and reported as service `release-manager`.
The jaeger configuration can be customized with available [environment variables](https://github.com/jaegertracing/jaeger-client-go#environment-variables).

For local development a jaeger all-in-one instance can be created with Docker running `make jaeger`.
The Jaeger UI will be available on [`localhost:16686`](http://localhost:16686).

To disable collection set `JAEGER_DISABLED=true`.

# Storage

Multiple entities are stored in different storage layers to allow the release manager to work.

## Artifacts

Artifacts are stored in an AWS S3 bucket.
They are stored as `zip` files with keys from the service name and artifact id.

```
.
├── <service>
│  └── <artifact-id>
└── example
    └── master-sha1234-plan1234
```

## Policies

Policies are stored in the Git repository along with all releases.
Each service policy is a JSON file in the `policies/<service>.json` path.

```json
{
  "service": "example",
  "autoReleases": [
    {
      "id": "auto-release-master-dev",
      "branch": "master",
      "environment": "dev"
    }
  ]
}
```

# Installation

All the applications are cross compiled to Linux and MacOS and available in the [Releases](https://github.com/lunarway/release-manager/releases) page.
The server and daemon are also available as Docker images at [quay.io/lunarway/release-manager](quay.io/lunarway/release-manager) and [quay.io/lunarway/release-daemon](quay.io/lunarway/release-daemon).

Usually you will release the server and daemon with kubernetes `Deployment` resources and distribute the `hamctl` CLI to developers.
`artifact` should be distributed to the Jenkins CI server and used in the pipelines.

## Access to the config repository

The release manager server needs read/write permissions to the config repo.

To create a secret that the release manager can consume: (expects that the filename is identity)

```
kubectl create secret generic release-manager-git-deploy --from-file=identity=key
```

This secret should be mounted to `/etc/release-manager/ssh`

# Development

The `Makefile` exposes targets for building, testing and deploying the release manager and its CLIs.
See it for details.

The most common operations are build and tests.

```
$ make build
go build -o dist/hamctl ./cmd/hamctl
go build -o dist/server ./cmd/server
go build -o dist/artifact ./cmd/artifact

$ make build_server
go build -o dist/server ./cmd/server

$ make test
go test -v ./...
```

## e2e setup

To help development it is possible to use the e2e setup.

This setup is based a kubernetes cluster managed by `kind`. The following resources is setup up

| Name              | Description                                                                                                                                                                                                                                                                                                  |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `source-git-repo` | A local git repository in `e2e-test/source-git-repo` that is used as the config repository.                                                                                                                                                                                                                  |
| `fluxd`           | The fluxd service inside the k8s cluster, which is connected to the `source-git-repo`. It is polling the repo for changes every 5s, so it triggers as soon as a commit is done in `source-git-repo`, like a webhook from github normally would. Additionally fluxd is setup to `--connect` to release-daemon |
| `release-daemon`  | A locally built binary of the release-daemon, but running inside the k8s cluster. The binary is mounted from local `e2e-test/binaries` for quick rebuild, so the pod can just be restarted while developing. This is done using the **rebuild** or **watch** actions.                                        |
| `release-server`  | A locally built binary of the release-daemon, that is running in the same manner as the `release-daemon`                                                                                                                                                                                                     |
| `rabbitmq`        | A simply setup rabbitmq server for the release-manager                                                                                                                                                                                                                                                       |

To use the e2e setup there are the following actions supported:

| Action                   | Command                          | Description                                                                                          |
| ------------------------ | -------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Start e2e setup          | `make e2e-setup`                 | Start and initiate kind and e2e setup                                                                |
| Rebuild manager          | `make e2e-rebuild-local-manager` | Rebuild the manager and restart pod in e2e cluster                                                   |
| Rebuild daemon           | `make e2e-rebuild-local-daemon`  | Like "Rebuild manager" but for the daemon                                                            |
| Watch manager            | `make e2e-rebuild-local-manager` | Watch source code changes and rebuild the manager and restart pod in e2e cluster. Requires `nodemon` |
| Watch daemon             | `make e2e-rebuild-local-daemon`  | Like "Watch manager" but for the daemon                                                              |
| Do dummy release         | `make e2e-do-release`            | Do a release in git repo to trigger fluxd change                                                     |
| Do another dummy release | `make e2e-do-another-release`    | Do another kind of release in git repo to trigger fluxd change                                       |
| Stop e2e setup           | `make e2e-teardown`              | Stop and cleanup the e2e setup                                                                       |

## Releasing

This project is configured with `goreleaser` and releases all 4 applications at once.
Push a new tag to the main branch and GitHub actions will publish a new release and create the changelog.
