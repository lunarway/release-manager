example: 
	- rm -rf examples/.artifact.json
	go run main.go init \
		--git-author "Kasper Nissen"\
		--git-message "Some message"\
		--git-sha "asd39sdas0g392"\
		--git-committer "Bjørn Sørensen"\
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
		--tag "master-1234ds"\
		--docker-version "1.18.6"\
		--root examples
	
	go run main.go add push\
		--image "quay.io/lunarway/application"\
		--tag "master-1234ds"\
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

	
