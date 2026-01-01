test:
	go mod tidy
	go test -race -v ./...
	