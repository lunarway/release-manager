example:
	rm -rf examples
	mkdir -p  examples
	go run main.go init \
	  --artifact-id "master-1234ds13g3-12s46g356g"\
		--git-author-name "Kasper Nissen"\
		--git-author-email "kni@lunarway.com"\
		--git-message "Some message"\
		--git-sha "asd39sdas0g392"\
		--git-committer-name "Bjørn Sørensen"\
		--git-committer-email "bso@lunarway.com"\
		--provider "BitBucket"\
		--url "https://someurl.com"\
		--name "lunar-way-application"\
		--shuttle-plan-sha "asdasdo300asd0asd90as92"\
		--shuttle-plan-url "https://someplanurl"\
    --shuttle-plan-message "Some commit"\
		--ci-job-url "https://jenkins.dev.lunarway.com/job/asdasd"\
		--root examples

	go run main.go add build\
		--image "quay.io/lunarway/application"\
		--tag "master-1234ds13g3-12s46g356g"\
		--docker-version "1.18.6"\
		--root examples

	go run main.go add push\
		--image "quay.io/lunarway/application"\
		--tag "master-1234ds13g3-12s46g356g"\
		--docker-version "1.18.6"\
		--root examples

	go run main.go add test\
		--url "https://jenkins.dev.lunarway.com"\
		--passed 563\
		--skipped 0\
		--failed 0\
		--root examples

	go run main.go add snyk-code\
		--language "go"\
		--snyk-version "1.144.23"\
		--url "https://snyk.io/aslkdasdlas"\
		--high 2\
		--medium 23\
		--low 134\
		--root examples

	go run main.go add snyk-docker\
		--base-image "node"\
		--snyk-version "1.144.23"\
		--tag "8.15.0-alpine"\
		--url "https://snyk.io/aslkdasdlas"\
		--high 0\
		--medium 0\
		--low 0\
		--root examples

	go run main.go end --root examples

define CONFIG_MAP_DEV
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
export CONFIG_MAP_DEV

example_resources:
	mkdir -p examples/dev
	echo "$$CONFIG_MAP_DEV" > examples/dev/configmap.yaml

test_push: example example_resources
	go run main.go push \
		--root examples \
		--config-repo git@github.com:lunarway/release-manager-test-config-repo.git \
		--git-branch master \
		--service test \
		--ssh-private-key ~/.ssh/github_key
