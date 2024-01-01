.PHONY: setup test format

setup: 
	@cp githooks/* .git/hooks
	@ls githooks | xargs -I {} chmod +x .git/hooks/{}

test:
	@go test -v ./...

format: 
	@go fmt ./...
	@golangci-lint run ./...
