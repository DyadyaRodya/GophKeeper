LOCAL_BIN:=$(CURDIR)/bin

.PHONY: install_bin
install_bin: # install binary dependencies
	mkdir -p $(LOCAL_BIN)
	GOBIN=$(LOCAL_BIN) go mod tidy
	GOBIN=$(LOCAL_BIN) go install github.com/vektra/mockery/v2@latest
	GOBIN=$(LOCAL_BIN) go install golang.org/x/tools/cmd/goimports@latest
	GOBIN=$(LOCAL_BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

.PHONY: install
install: install_bin

.PHONY: generate
generate: # generate gRPC files
	protoc --experimental_allow_proto3_optional --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/v1/gophkeeper.proto

.PHONY: lint
lint: # run statictest
	$(LOCAL_BIN)/goimports -local "github.com/DyadyaRodya/GophKeeper" -w cmd internal pkg
	#go vet -vettool=/usr/bin/statictest ./... # broken for go 1.23
	go build -o cmd/staticlint/main cmd/staticlint/main.go && go vet -vettool=cmd/staticlint/main ./internal/... ./pkg/... ./cmd/...

.PHONY: tests
tests: # run unit tests
	go test -race -coverprofile=coverage.out ./...

.PHONY: build-server
build-server: # build example
	go build -ldflags="-X main.buildVersion=v${VERSION} -X 'main.buildDate=$$(date +'%Y/%m/%d %H:%M:%S')' -X 'main.buildCommit=$$(git rev-parse --short HEAD)'" -o cmd/gophkeeperserver/main cmd/gophkeeperserver/main.go

.PHONY: build-client
build-client: # build example
	go build -ldflags="-X main.buildVersion=v${VERSION} -X 'main.buildDate=$$(date +'%Y/%m/%d %H:%M:%S')' -X 'main.buildCommit=$$(git rev-parse --short HEAD)'" -o cmd/gophkeeperclient/main cmd/gophkeeperclient/main.go
