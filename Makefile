.PHONY: setup test build format

setup: 
	@cp githooks/* .git/hooks
	@ls githooks | xargs -I {} chmod +x .git/hooks/{}

test: 
ifeq ($(verbose), true) 
	@go test -v ./...
else
	@go test ./...
endif

release: ./scripts/makefile-release.sh
ifndef version
	@./scripts/makefile-release.sh $(version)
endif

format: 
	@go fmt ./...
	@golangci-lint run ./...
