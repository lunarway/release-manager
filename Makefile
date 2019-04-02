.DEFAULT: build
build: build_hamctl build_server build_artifact build_daemon

build_artifact:
	go build -o dist/artifact ./cmd/artifact

build_hamctl:
	go build -o dist/hamctl ./cmd/hamctl

build_server:
	go build -o dist/server ./cmd/server

build_daemon:
	go build -o dist/daemon ./cmd/daemon

build_daemon_docker:
ifeq ($(TAG),)
	@echo "TAG is required for this target" && exit 1
endif
	docker build -f Dockerfile-daemon -t quay.io/lunarway/release-daemon:${TAG} .

push_daemon_docker:
ifeq ($(TAG),)
	@echo "TAG is required for this target" && exit 1
endif
	docker push quay.io/lunarway/release-daemon:${TAG}

IMAGE=quay.io/lunarway/release-manager
build_server_docker:
ifeq ($(TAG),)
	@echo "TAG is required for this target" && exit 1
endif
	docker build -f Dockerfile-server -t $(IMAGE):${TAG} .

push_server_docker:
ifeq ($(TAG),)
	@echo "TAG is required for this target" && exit 1
endif
	docker push ${IMAGE}:${TAG}

test:
	go test -v ./...

server: build_server
	HAMCTL_AUTH_TOKEN=test DAEMON_AUTH_TOKEN=test ./dist/server start --ssh-private-key ~/.ssh/github --slack-token ${SLACK_TOKEN}

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

# posts a github push webhook to localhost:8080 for a product build commit
github-webhook:
	curl -H 'X-GitHub-Event: push' \
	-d '{ \
		"ref": "refs/heads/master", \
		"head_commit": { \
			"id": "sha", \
			"message": "[product] build something", \
			"modified": [ \
				"builds/product/master/artifact.json", \
				"builds/product/master/dev/40-deployment.yaml", \
				"builds/product/master/prod/40-deployment.yaml", \
				"builds/product/master/staging/40-deployment.yaml" \
			] \
		} \
	}' \
	localhost:8080/webhook/github

daemon-webhook-success:
	curl -X POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer test" \
	-d '{ \
	  "name": "product-f4fd84588-62789", \
	  "namespace": "dev", \
	  "state": "Running", \
	  "artifactId": "master-a9aad46188-f41b35775e", \
	  "reason": "test", \
	  "message": "test", \
	  "containers": [ \
		{ "name": "container1", "state": "Running" }, \
		{ "name": "container2", "state": "Running" } \
	  ] \
	}' \
	localhost:8080/webhook/daemon

daemon-webhook-crashloop:
	curl -X POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer test" \
	-d '{ \
	  "name": "product-f4fd84588-62789", \
	  "namespace": "dev", \
	  "state": "CrashLoopBackOff", \
	  "artifactId": "master-a9aad46188-f41b35775e", \
	  "reason": "CrashLoopBackOff", \
	  "message": "test", \
	  "logs": "some error logs here", \
	  "containers": [ \
		{ "name": "container1", "state": "CrashLoopBackOff" }, \
		{ "name": "container2", "state": "Running" } \
	  ] \
	}' \
	localhost:8080/webhook/daemon

daemon-webhook-configerror:
	curl -X POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer test" \
	-d '{ \
	  "name": "product-f4fd84588-62789", \
	  "namespace": "dev", \
	  "state": "CreateContainerConfigError", \
	  "artifactId": "master-a9aad46188-f41b35775e", \
	  "reason": "CreateContainerConfigError", \
	  "message": "Config error. 'secret/log.debug' not set", \
	  "containers": [ \
		{ "name": "container1", "state": "CreateContainerConfigError" }, \
		{ "name": "container2", "state": "Running" } \
	  ] \
	}' \
	localhost:8080/webhook/daemon
