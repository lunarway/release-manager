# Release Manager
GitOps release manager for kubernetes configuration reposistories.

## Policies

It is possible to configure policies for releases with `hamctl`'s `policy` command.

You can `add`, `remove` and `list` policies for a specific service like below.

```
$ hamctl policy --service <service> list
$ hamctl policy --service <service> add <policy>
$ hamctl policy --service <service> remove <policy-id> [<policy-id>]
```

See below for details on how to add specific policies.

### Auto-release artifacts from branch to environments

An `auto-release` policy instructs the release manager to deploy new artifacts from a specific branch into an environment.

Multiple policies can be added for the same branch to different environments, e.g. release `master` artifacts to `dev` and `staging`.

This is an example of adding an auto-release policy for the product service for the `master` branch and `dev` environment.

```
$ hamctl policy --service product add auto-release --branch master --env dev
```

## Access to the config repository
The release manager needs read/write permissions to the config repo.

To create a secret that the release manager can consume: (expects that the filename is identity)

```
kubectl create secret generic release-manager-git-deploy --from-file=identity=key
```

This secret should be mounted to `/etc/release-manager/ssh`

# Development

The `Makefile` exposes targets for building, testing and deploying the release manager.
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
