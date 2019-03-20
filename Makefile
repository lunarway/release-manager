.DEFAULT: build
build: build_hamctl build_server build_artifact

build_artifact:
	go build -o dist/artifact ./cmd/artifact

build_hamctl:
	go build -o dist/hamctl ./cmd/hamctl

build_server:
	go build -o dist/server ./cmd/server

IMAGE=quay.io/lunarway/release-manager
build_server_docker:
ifeq ($(TAG),)
	@echo "TAG is required for this target" && exit 1
endif
	docker build -t $(IMAGE):${TAG} .

push_docker:
ifeq ($(TAG),)
	@echo "TAG is required for this target" && exit 1
endif
	docker push ${IMAGE}:${TAG}

test:
	go test -v ./...

server: build_server
	RELEASE_MANAGER_AUTH_TOKEN=test ./dist/server start --ssh-private-key ~/.ssh/github

release:
	goreleaser --rm-dist --skip-publish

deploy: deploy-jenkins-dev deploy-jenkins-prod

deploy-jenkins-dev:
	GOOS=linux GOARCH=amd64 go build -o artifact-linux-amd64 cmd/artifact/main.go
	scp artifact-linux-amd64 lunar-dev-jenkins:/usr/local/bin/artifact

deploy-jenkins-prod:
	GOOS=linux GOARCH=amd64 go build -o artifact-linux-amd64 cmd/artifact/main.go
	scp artifact-linux-amd64 lunar-prod-jenkins:/usr/local/bin/artifact

install-hamctl: build_hamctl
	chmod +x cmd/hamctl
	cp dist/hamctl /usr/local/bin/hamctl
