
tidy:
	go mod tidy

generate:
	go generate ./...

vet: tidy generate
	go vet ./...

fmt: tidy
	go fmt ./...
