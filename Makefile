.PHONY: setup test format

setup: 
	@cp githooks/* .git/hooks
	@ls githooks | xargs -I {} chmod +x .git/hooks/{}

test: 
ifeq ($(verbose), true) 
	@go test -v ./...
else
	@go test ./...
endif

format: 
	@go fmt ./...
	@golangci-lint run ./...
