example:
	rm -rf examples
	mkdir -p  examples
	go build
	./artifact init \
		--service "test-service"\
	  --artifact-id "master-1234ds13g3-12s46g356g"\
		--git-branch "master"\
		--git-author-name "Kasper Nissen"\
		--git-author-email "kni@lunar.app"\
		--git-message "Some message"\
		--git-sha "asd39sdas0g392"\
		--git-committer-name "Bjørn Sørensen"\
		--git-committer-email "bso@lunar.app"\
		--provider "BitBucket"\
		--url "https://someurl.com"\
		--name "lunar-way-application"\
		--shuttle-plan-sha "asdasdo300asd0asd90as92"\
		--shuttle-plan-url "https://someplanurl"\
    --shuttle-plan-message "Some commit"\
		--shuttle-plan-branch "plan-branch"\
		--ci-job-url "https://jenkins.dev.lunarway.com/job/asdasd"\
		--root examples

	./artifact add build\
		--image "quay.io/lunarway/application"\
		--tag "master-1234ds13g3-12s46g356g"\
		--docker-version "1.18.6"\
		--root examples

	./artifact add push\
		--image "quay.io/lunarway/application"\
		--tag "master-1234ds13g3-12s46g356g"\
		--docker-version "1.18.6"\
		--root examples

	./artifact add test\
		--url "https://jenkins.dev.lunarway.com"\
		--passed 563\
		--skipped 0\
		--failed 0\
		--root examples

	./artifact add snyk-code\
		--language "go"\
		--snyk-version "1.144.23"\
		--url "https://snyk.io/aslkdasdlas"\
		--high 2\
		--medium 23\
		--low 134\
		--root examples

	./artifact add snyk-docker\
		--base-image "node"\
		--snyk-version "1.144.23"\
		--tag "8.15.0-alpine"\
		--url "https://snyk.io/aslkdasdlas"\
		--high 0\
		--medium 0\
		--low 0\
		--root examples

	./artifact end --root examples

define CONFIG_MAP
apiVersion: v1
kind: ConfigMap
metadata:
  name: product
  namespace: dev
data:
  cdn.url: 'https://cdn.dev.lunarway.com'
  log.console.as.json: 'true'
  log.console.level: 'info'
endef
export CONFIG_MAP

example_resources:
	mkdir -p examples/{dev,staging,prod}
	echo "$$CONFIG_MAP" > examples/dev/configmap.yaml
	echo "$$CONFIG_MAP" > examples/staging/configmap.yaml
	echo "$$CONFIG_MAP" > examples/prod/configmap.yaml

SSH_PRIVATE_KEY=~/.ssh/github
RELEASE_MANAGER_URL=http://localhost:8080
RELEASE_MANAGER_AUTH_TOKEN=test
CONFIG_REPO=git@github.com:lunarway/release-manager-test-config-repo.git

test_push: example example_resources
	./artifact push \
		--root examples \
		--http-base-url ${RELEASE_MANAGER_URL} \
		--http-auth-token ${RELEASE_MANAGER_AUTH_TOKEN} \
		--config-repo="${CONFIG_REPO}" \
		--ssh-private-key=${SSH_PRIVATE_KEY}
