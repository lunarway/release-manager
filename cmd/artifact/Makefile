ARTIFACT_ID=master-`date +"%Y%m%d"`-`date +"%H%M"`

example:
	rm -rf examples
	mkdir -p  examples
	go build
	./artifact init \
		--service "test-service"\
		--artifact-id "${ARTIFACT_ID}"\
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
		--ci-job-url "https://jenkins.corp.com/job/asdasd"\
		--root examples

	./artifact add build\
		--image "quay.io/lunarway/application"\
		--tag "${ARTIFACT_ID}"\
		--docker-version "1.18.6"\
		--root examples

	./artifact add push\
		--image "quay.io/lunarway/application"\
		--tag "${ARTIFACT_ID}"\
		--docker-version "1.18.6"\
		--root examples

	./artifact add test\
		--url "https://jenkins.corp.com"\
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
annotations:
  lunar.tech/artifact-id: ${ARTIFACT_ID}
data:
  cdn.url: 'https://cdn.dev.lunarway.com'
  log.console.as.json: 'true'
  log.console.level: 'info'
endef
export CONFIG_MAP

define FLUX_KUSTOMIZATION
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: product
  namespace: dev
spec:
  interval: 5m0s
  path: ./dev/releases/dev/product
  prune: true
  sourceRef:
    kind: GitRepository
    name: k8s-cluster-config
endef
export FLUX_KUSTOMIZATION

example_resources:
	mkdir -p examples/{dev,prod}
	echo "$$CONFIG_MAP" > examples/dev/configmap.yaml
	echo "$$CONFIG_MAP" > examples/prod/configmap.yaml
	echo "$$FLUX_KUSTOMIZATION" > examples/dev/kustomization.yaml
	echo "$$FLUX_KUSTOMIZATION" > examples/prod/kustomization.yaml

RELEASE_MANAGER_URL=http://localhost:8080
OAUTH_IDP_URL=https://idpurl
OAUTH_CLIENT_ID=id
OAUTH_CLIENT_SECRET=secret

test_push: example example_resources
	./artifact push \
		--root examples \
		--http-base-url ${RELEASE_MANAGER_URL} \
		--client-id ${OAUTH_CLIENT_ID} \
		--client-secret ${OAUTH_CLIENT_SECRET} \
		--idp-url ${OAUTH_IDP_URL}
