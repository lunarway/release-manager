release:
	goreleaser --rm-dist --skip-publish

deploy-jenkins-dev: 
	GOOS=linux GOARCH=amd64 go build -o rm_artifact-linux-amd64 cmd/rm_artifact/main.go 
	scp rm_artifact-linux-amd64 lunar-dev-jenkins:/usr/local/bin/rm_artifact

deploy-jenkins-prod: 
	GOOS=linux GOARCH=amd64 go build -o rm_artifact-linux-amd64 cmd/rm_artifact/main.go 
	scp rm_artifact-linux-amd64 lunar-prod-jenkins:/usr/local/bin/rm_artifact

deploy: deploy-jenkins-dev deploy-jenkins-prod
	