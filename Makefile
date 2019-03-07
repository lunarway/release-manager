release:
	goreleaser --rm-dist --skip-publish

deploy-jenkins-dev: 
	GOOS=linux GOARCH=amd64 go build -o artifact-linux-amd64 cmd/artifact/main.go 
	scp artifact-linux-amd64 lunar-dev-jenkins:/usr/local/bin/artifact

deploy-jenkins-prod: 
	GOOS=linux GOARCH=amd64 go build -o artifact-linux-amd64 cmd/artifact/main.go 
	scp artifact-linux-amd64 lunar-prod-jenkins:/usr/local/bin/artifact

deploy: deploy-jenkins-dev deploy-jenkins-prod
	
generate-go:
	- mkdir -p generated/grpc
	docker run --rm -v $(shell pwd):$(shell pwd) -w $(shell pwd) znly/protoc -I. protos/*.proto --go_out=plugins=grpc:.
	mv protos/*.pb.go generated/grpc/

server: 
	go build -o dist/server ./cmd/server
	./dist/server start

hamctl: 
	go build -o dist/hamctl ./cmd/hamctl
	./dist/hamctl help

artifact: 
	go build -o dist/artifact ./cmd/artifact
	./dist/artifact help