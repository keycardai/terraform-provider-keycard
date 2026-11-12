default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v .

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
	go test -v -cover -timeout=120s -parallel=1 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout=500s -parallel=5 ./...

# Deletes tftest-* zones older than an hour in the org of the KEYCARD_* credentials.
# SWEEP_DRY_RUN=1 lists them without deleting; check that first when running it by hand.
SWEEP_DRY_RUN ?= false
sweep:
	go run ./cmd/sweep -dry-run=$(SWEEP_DRY_RUN)

.PHONY: fmt lint test testacc sweep build install generate-client generate-docs generate
