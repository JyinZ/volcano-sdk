
run:
	go run .

tidy:
	go mod tidy

generate:
	go generate ./...

vet: tidy generate
	go vet ./...

fmt: tidy
	go fmt ./...

build: tidy
	go build -v .

build-linux: tidy
	GOOS="linux" GOARCH=amd64 \
		go build -v .

build-win: tidy
	GOOS="windows" go build -v .

install: tidy
	go install -v ./...

.PHONY: run install