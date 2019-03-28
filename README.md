# Release Manager
GitOps release manager for kubernetes configuration reposistories.

## Policies

It is possible to configure policies for releases with `hamctl`'s `policy` command.

You can `list`, `apply` and `delete` policies for a specific service like below.

```
$ hamctl policy --service <service> list
$ hamctl policy --service <service> apply <policy>
$ hamctl policy --service <service> delete <policy-id> [<policy-id>]
```

See below for details on how to apply specific policies.

### Auto-release artifacts from branches to environments

An `auto-release` policy instructs the release manager to deploy new artifacts from a specific branch into an environment.

Multiple policies can be applied for the same branch to different environments, e.g. release `master` artifacts to `dev` and `staging`.

This is an example of applying an auto-release policy for the product service for the `master` branch and `dev` environment.

```
$ hamctl policy --service product apply auto-release --branch master --env dev
```

# Design

The applications are basically utilities for moving files around a Git repository.
The release manager is a server that can be instructed through an HTTP API to perform certain actions, e.g. promote a release, release an artifact.

`hamctl` is a CLI for interacting with the server and `artifact` is a CLI for generating an artifact specification.

## Directory structure

Files are structured as shown below.

Artifacts are stored in the `builds` directory.
It contains artifacts based of Git branches on the application repositories and must contain resource definitions for the environments that it is able to be released to.

In the root are folders for each environment, e.g. `dev`, `prod`.
These folders contain a `releases` directory with kubernetes resource definitions of each namespace and their running applications.

A `policies` directory holds all recorded release policies.
These are stored as JSON files for each service.

```
.
├── policies
│   └── <service>.json
├── builds
│   └── <service>
│       ├── <branches>
│       └── master
│           ├── artifact.json
│           ├── <environment>
│           └── dev
│               ├── 01-configmap.yaml
│               ├── 02-db-configmap.yaml
│               ├── 40-deployment.yaml
│               └── 50-service.yaml
├── <environments>
└── dev
    ├── provisioning
    └── releases
        ├── <namespaces>
        └── dev
            └── <service>
                ├── artifact.json
                ├── 01-configmap.yaml
                ├── 02-db-configmap.yaml
                ├── 40-deployment.yaml
                └── 50-service.yaml
```

# Installation

## Access to the config repository
The release manager needs read/write permissions to the config repo.

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

# Release

There are multiple applications in this repo.
Only the server can be released to Docker as of now.

## Server

To build and push a new version of the release manager use the `build_server_docker` and `push_docker` targets.

```
$ make build_server_docker TAG=v1.2.3
$ make push_server_docker TAG=v1.2.3
```
