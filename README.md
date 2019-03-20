# Release Manager
GitOps release manager for kubernetes configuration reposistories.


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
$ make push_docker TAG=v1.2.3
```
