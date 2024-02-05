.PHONY: setup test test_coverage build format

setup: 
	@cp githooks/* .git/hooks
	@ls githooks | xargs -I {} chmod +x .git/hooks/{}

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
