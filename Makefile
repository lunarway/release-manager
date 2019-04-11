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

artifact-init:
	./dist/artifact init --slack-token ${SLACK_TOKEN} --artifact-id "master-deed62270f-854d930ecb" --name "lunar-way-product-service" --service "product" --git-author-name "Kasper Nissen" --git-author-email "kni@lunarway.com" --git-message "This is a test message" --git-committer-name "Bjørn Sørensen" --git-committer-email "bso@lunarway.com" --git-sha deed62270f24f1ca8cf2c19b505b2c88036e1b1c --git-branch master --url "https://bitbucket.org/LunarWay/lunar-way-product-service/commits/a05e314599a7c202724d46a009fcc0f493bce035" --ci-job-url "https://jenkins.dev.lunarway.com/job/bitbucket/job/lunar-way-product-service/job/master/170/display/redirect"

artifact-test:
	./dist/artifact add test --slack-token ${SLACK_TOKEN} --passed 189 --failed 0 --skipped 0

artifact-build:
	./dist/artifact add build --slack-token ${SLACK_TOKEN} --image quay.io/lunarway/product-service --tag master-24sadj821s-99sie2j19k --docker-version 1.18.09

artifact-push:
	./dist/artifact add push --slack-token ${SLACK_TOKEN} --image quay.io/lunarway/product-service --tag master-24sadj821s-99sie2j19k --docker-version 1.18.09

artifact-snyk-docker:
	./dist/artifact add snyk-docker --slack-token ${SLACK_TOKEN} --high 1 --medium 2 --low 23 --url ""

artifact-snyk-code:
	./dist/artifact add snyk-code --slack-token ${SLACK_TOKEN} --high 0 --medium 0 --low 0 --url "https://example.com"

artifact-failure:
	./dist/artifact failure --slack-token ${SLACK_TOKEN}

artifact-successful:
	./dist/artifact successful --slack-token ${SLACK_TOKEN}

artifact-slack: build_artifact artifact-init artifact-build artifact-test artifact-snyk-docker artifact-snyk-code artifact-push artifact-failure

release:
	goreleaser --rm-dist --skip-publish


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
				"artifacts/product/master/artifact.json", \
				"artifacts/product/master/dev/40-deployment.yaml", \
				"artifacts/product/master/prod/40-deployment.yaml", \
				"artifacts/product/master/staging/40-deployment.yaml" \
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
	  "state": "Ready", \
	  "artifactId": "master-a9aad46188-f41b35775e", \
	  "reason": "test", \
	  "message": "test", \
	  "containers": [ \
		{ "name": "container1", "state": "Ready" }, \
		{ "name": "container2", "state": "Ready" } \
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
