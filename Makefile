.DEFAULT: build

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir := $(dir $(mkfile_path))

build: build_hamctl build_server build_artifact build_daemon

VERSION=$(shell git rev-parse HEAD)
GO_BUILD=go build -ldflags='-s -w -X main.version=$(VERSION)'

build_artifact:
	${GO_BUILD} -o dist/artifact ./cmd/artifact

build_hamctl:
	${GO_BUILD} -o dist/hamctl ./cmd/hamctl

build_server:
	${GO_BUILD} -o dist/server ./cmd/server

build_daemon:
	${GO_BUILD} -o dist/daemon ./cmd/daemon

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

generate_mock:
	mockery --dir ./internal/policy --inpackage --name GitService
	mockery --dir ./internal/flow --inpackage --name GitService
	mockery --dir ./internal/slack --inpackage --name SlackClient
	# update imports of the slack client as it is not correctly resolved by
	# mockery on generation
	goimports -w ./internal/slack/mock_SlackClient.go

integration-test: rabbitmq-background
	@echo "Running integration tests against RabbitMQ on localhost"
ifneq ($(VERBOSE),)
	RELEASE_MANAGER_INTEGRATION_RABBITMQ_HOST=localhost go test -count=1 -v ./...
else
	RELEASE_MANAGER_INTEGRATION_RABBITMQ_HOST=localhost go test -count=1 ./...
endif


AUTH_TOKEN=test
SSH_PRIVATE_KEY=~/.ssh/github
CONFIG_REPO=git@github.com:lunarway/release-manager-test-config-repo.git
GRAFANA_URL=localhost
GRAFANA_API_KEY=grafana-api-key
SLACK_TOKEN=slack-token
USER_MAPPINGS=
BRANCH_RESTRICTIONS=
S3_BUCKET=release-manager-test

SERVER_START=./dist/server start \
		--ssh-private-key ${SSH_PRIVATE_KEY} \
		--slack-token ${SLACK_TOKEN} \
		--grafana-api-key-dev ${GRAFANA_API_KEY} \
		--grafana-url-dev ${GRAFANA_URL} \
		--hamctl-auth-token ${AUTH_TOKEN} \
		--daemon-auth-token ${AUTH_TOKEN} \
		--artifact-auth-token ${AUTH_TOKEN} \
		--s3-artifact-storage-bucket-name '${S3_BUCKET}' \
		--log.level debug \
		--log.development t \
		--config-repo ${CONFIG_REPO} \
		--user-mappings '${USER_MAPPINGS}' \
		--policy-branch-restrictions '${BRANCH_RESTRICTIONS}'

server-memory: build_server
		$(SERVER_START) \
		--broker-type memory

AMQP_USER=lunar
AMQP_PASSWORD=lunar

server-rabbitmq: build_server
		$(SERVER_START) \
		--broker-type amqp \
		--amqp-user ${AMQP_USER} \
		--amqp-password ${AMQP_PASSWORD}

artifact-init:
	USER_MAPPINGS="kaspernissen@gmail.com=kni@lunar.app,something@gmail.com=some@lunar.app" ./dist/artifact init --slack-token ${SLACK_TOKEN} --artifact-id "master-deed62270f-854d930ecb" --name "lunar-way-product-service" --service "product" --git-author-name "Kasper Nissen" --git-author-email "kaspernissen@gmail.com" --git-message "This is a test message" --git-committer-name "Bjørn Sørensen" --git-committer-email "test@gmail.com" --git-sha deed62270f24f1ca8cf2c19b505b2c88036e1b1c --git-branch master --url "https://bitbucket.org/LunarWay/lunar-way-product-service/commits/a05e314599a7c202724d46a009fcc0f493bce035" --ci-job-url "https://jenkins.corp.com/job/bitbucket/job/lunar-way-product-service/job/master/170/display/redirect"

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
	./dist/artifact failure --slack-token ${SLACK_TOKEN} --error-message "Build failed"

artifact-successful:
	./dist/artifact successful --slack-token ${SLACK_TOKEN}

artifact-slack: build_artifact artifact-init artifact-build artifact-test artifact-snyk-docker artifact-snyk-code artifact-push artifact-failure

release:
	goreleaser --rm-dist --skip-publish


install-hamctl: build_hamctl
	chmod +x cmd/hamctl
	cp dist/hamctl /usr/local/bin/hamctl

# url to a running release-manager
URL=localhost:8080

# posts a github push webhook to $(URL) for a product build commit
github-webhook:
	curl -H 'X-GitHub-Event: push' \
	-d '{ \
		"ref": "refs/heads/master", \
		"head_commit": { \
			"id": "sha", \
			"message": "[product] artifact master-1234ds13g3-12s46g356g by Foo Bar\nArtifact-created-by: Foo Bar <test@lunar.app>" \
		} \
	}' \
	$(URL)/webhook/github

hamctl-status:
	curl -X GET \
	-H 'Content-Type: application/json' \
	-H 'Authorization: Bearer test' \
	"$(URL)/status?service=a"

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
	$(URL)/webhook/daemon

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
	$(URL)/webhook/daemon

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
	$(URL)/webhook/daemon

server-profile-heap:
	mkdir -p profiles
	curl -o profiles/heap.pprof $(URL)/debug/pprof/heap
	go tool pprof -http=:8081 profiles/heap.pprof

server-profile-cpu:
	mkdir -p profiles
	curl -o profiles/cpu.pprof $(URL)/debug/pprof/profile?seconds=10
	go tool pprof -http=:8081 profiles/cpu.pprof

jaeger:
	open http://localhost:16686
	docker run --rm -p 5775:5775/udp -p 6831:6831/udp -p 6832:6832/udp -p 5778:5778 -p 16686:16686 -p 14268:14268 -p 9411:9411 -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 jaegertracing/all-in-one:1.7

RABBITMQ_INTEGRATION_HOST_CONTAINER=rm-rabbitmq

rabbitmq-background:
	@echo "Starting RabbitMQ in background"
	-docker start ${RABBITMQ_INTEGRATION_HOST_CONTAINER} 2>/dev/null || docker run --rm --hostname rabbitmq -p 5672:5672 -p 15672:15672 -e RABBITMQ_DEFAULT_USER=lunar -e RABBITMQ_DEFAULT_PASS=lunar --name ${RABBITMQ_INTEGRATION_HOST_CONTAINER} -d rabbitmq:3-management

rabbitmq-background-stop:
	@echo "Stopping RabbitMQ in background"
	-docker kill ${RABBITMQ_INTEGRATION_HOST_CONTAINER}

rabbitmq:
	@echo "Starting RabbitMQ. See admin dashboard on http://localhost:15672"
	docker run --rm --hostname rabbitmq -p 5672:5672 -p 15672:15672 -e RABBITMQ_DEFAULT_USER=lunar -e RABBITMQ_DEFAULT_PASS=lunar rabbitmq:3-management

e2e-setup: e2e-setup-git e2e-setup-kind e2e-setup-rabbitmq e2e-setup-manager e2e-setup-daemon e2e-setup-fluxd

	@echo "\nSetup complete\n\nRun the following to continue:\n\
	- make e2e-do-release"

e2e-teardown:
	kind delete cluster

e2e-setup-git:
	rm -rf e2e-test/source-git-repo
	mkdir -p e2e-test/source-git-repo/local/releases/default
	echo "Hello World" > e2e-test/source-git-repo/README.md
	cd e2e-test/source-git-repo;\
	git init;\
	git add .;\
	git commit -m "Add readme"

e2e-setup-kind:
	kind create cluster --config e2e-test/kind-cluster.yaml

e2e-setup-rabbitmq:
	kubectl apply -f $(current_dir)e2e-test/rabbitmq.yaml

e2e-setup-daemon: e2e-build-local-daemon
	docker build -f $(current_dir)Dockerfile-daemon-goreleaser -t kind-release-daemon:local $(current_dir)e2e-test/binaries
	kind load docker-image kind-release-daemon:local
	kubectl apply -f $(current_dir)e2e-test/release-daemon.yaml

e2e-setup-manager: e2e-build-local-manager
	# copy ssh-config to satishfy Dockerfile-server-goreleaser
	cp $(current_dir)ssh_config $(current_dir)e2e-test/binaries/ssh_config
	docker build -f $(current_dir)Dockerfile-server-goreleaser -t kind-release-manager:local $(current_dir)e2e-test/binaries
	kind load docker-image kind-release-manager:local
	kubectl apply -f $(current_dir)e2e-test/release-manager.yaml

e2e-setup-fluxd:
	kubectl apply -f e2e-test/fluxd.yaml

e2e-build-local-daemon:
	mkdir -p $(current_dir)e2e-test/binaries
	env GOOS=linux go build -o $(current_dir)e2e-test/binaries/daemon ./cmd/daemon

e2e-rebuild-local-daemon: e2e-build-local-daemon
	kubectl delete pods -l app=release-daemon

e2e-watch-local-daemon:
	nodemon --ext .go --watch "cmd/daemon/**" --watch "internal/**" --exec "set -e; make e2e-rebuild-local-daemon; while true; do kubectl logs deploy/release-daemon -f || true; done;"

e2e-build-local-manager:
	mkdir -p $(current_dir)e2e-test/binaries
	env GOOS=linux go build -o $(current_dir)e2e-test/binaries/server ./cmd/server

e2e-rebuild-local-manager: e2e-build-local-manager
	kubectl delete pods -l app=release-manager

e2e-watch-local-manager:
	nodemon --ext .go --watch "cmd/server/**" --watch "internal/**" --exec "set -e; make e2e-rebuild-local-manager; while true; do kubectl logs deploy/release-manager -f || true; done;"

e2e-do-release:
	echo "apiVersion: v1\n\
kind: ConfigMap\n\
metadata:\n\
  name: test\n\
  namespace: default\n\
data:\n\
  somevalue: $$(date)" > ./e2e-test/source-git-repo/local/releases/default/test.yaml
	cd ./e2e-test/source-git-repo;\
	git add .;\
	git commit -m "[env/service] release master-5975d93540-ddcd22312b by kni@lunar.app"

e2e-do-another-release:
	echo "apiVersion: v1\n\
kind: ConfigMap\n\
metadata:\n\
  name: another-test\n\
  namespace: default\n\
data:\n\
  some-other-value: $$(date)" > ./e2e-test/source-git-repo/local/releases/default/another-test.yaml
	cd ./e2e-test/source-git-repo;\
	git add .;\
	git commit -m "[env/service] release master-5975d93540-ddcd22312b by kni@lunar.app"

e2e-do-duplicate-release:
	echo "apiVersion: v1\n\
kind: ConfigMap\n\
metadata:\n\
  name: test\n\
  namespace: default\n\
baddata:\n\
  somevalue: $$(date)" > ./e2e-test/source-git-repo/local/releases/default/test.yaml
	cd ./e2e-test/source-git-repo;\
	git add .;\
	git commit -m "[env/service] release master-5975d93540-ddcd22312b by kni@lunar.app"

	echo "apiVersion: v1\n\
kind: ConfigMap\n\
metadata:\n\
  name: test\n\
  namespace: default\n\
baddata:\n\
  somevalue: $$(date)" > ./e2e-test/source-git-repo/local/releases/default/dup.yaml
	cd ./e2e-test/source-git-repo;\
	git add .;\
	git commit -m "[env/service] release master-5975d93540-123456789 by kni@lunar.app"
