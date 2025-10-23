default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate-client:
	go generate ./...

generate-docs:
	cd tools; go generate ./...

generate: generate-docs generate-client

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120s -parallel=10 ./...

.PHONY: fmt lint test testacc build install generate-client generate-docs generate
