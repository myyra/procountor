default:
  @just --list

install-tools:
  go install github.com/ogen-go/ogen/cmd/ogen@latest

ogen-generate: 
  ogen -clean -config ogen.yaml -package procountorapi -target procountorapi internal/apispec/procountor-api.yaml

tidy:
  go mod tidy

generate: ogen-generate tidy

build:
  go build ./...

install:
  go install ./cmd/procountor

format:
  gofmt -w -l .

test:
  go test ./...

lint:
  golangci-lint run ./...
