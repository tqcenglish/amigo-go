all:
	go run -race examples/main.go
fmt:
	gofmt -w .