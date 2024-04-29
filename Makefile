.PHONY: setup setup_devdeps test test_coverage run_example build release format

setup: setup_devdeps
	@go mod download
	@cp githooks/* .git/hooks
	@ls githooks | xargs -I {} chmod +x .git/hooks/{}

setup_devdeps:
	@go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

test: 
ifeq ($(verbose), true) 
	@go test -v -cover ./...
else
	@go test -cover ./...
endif

test_coverage:
	@go test -coverprofile=c.out
ifdef save
	@go tool cover -html="c.out" -o coverage.html
else
	@go tool cover -html="c.out" && rm c.out
endif

run_example:
	@go run ./example

publish:
ifdef version
	go mod tidy
	@GOPROXY=proxy.golang.org go list -m github.com/atifcppprogrammer/yflicks-yts@v$(version)
else
	@echo please provide release version for yflicks-yts
endif

release:
ifdef version
	@git-chglog --next-tag v$(version) --output CHANGELOG.md
	@git add CHANGELOG.md
	@git commit -m "chore(release): v$(version)"
	@git tag -sa -m "yflicks-yts-$(version)" v$(version)
endif

format: 
	@go fmt ./...
	@golangci-lint run ./...
