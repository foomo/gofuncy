.DEFAULT_GOAL:=help
-include .makerc

# --- Config -----------------------------------------------------------------

# Newline hack for error output
define br


endef

# --- Targets -----------------------------------------------------------------

# This allows us to accept extra arguments
%: .mise .lefthook
	@:

.PHONY: .mise
# Install dependencies
.mise:
ifeq (, $(shell command -v mise))
	$(error $(br)$(br)Please ensure you have 'mise' installed and activated!$(br)$(br)  $$ brew update$(br)  $$ brew install mise$(br)$(br)See the documentation: https://mise.jdx.dev/getting-started.html)
endif
	@mise install

.PHONY: .lefthook
# Configure git hooks for lefthook
.lefthook:
	@lefthook install --reset-hooks-path

### Tasks

.PHONY: check
## Run lint & test
check: tidy generate lint.fix test.race test.bench audit

.PHONY: tidy
## Run go mod tidy
tidy:
	@echo "〉go mod tidy"
	@go mod tidy

.PHONY: lint
## Run linter
lint:
	@echo "〉golangci-lint run"
	@golangci-lint run

.PHONY: lint.fix
## Fix lint violations
lint.fix:
	@echo "〉golangci-lint run fix"
	@golangci-lint run --fix

.PHONY: lint.branch
## Run linter with --new-from-rev=origin/main
lint.branch:
	@echo "〉golangci-lint run with --new-from-rev=origin/main"
	@golangci-lint run --new-from-rev=origin/main

.PHONY: test
## Run tests
test:
	@echo "〉go test"
	@GO_TEST_TAGS=-skip go test -v -tags=safe -shuffle=on -coverprofile=coverage.out ./...

.PHONY: test.race
## Run tests with -race
test.race:
	@echo "〉go test -race"
	@GO_TEST_TAGS=-skip go test -v -tags=safe -shuffle=on -coverprofile=coverage.out -race ./...

.PHONY: test.bench
## Run tests with -bench
test.bench:
	@echo "〉go test -bench"
	@GO_TEST_TAGS=-skip go test -v -tags=safe -shuffle=on -bench=. -benchmem ./... > benchmarks.out

.PHONY: generate
## Run go generate
generate:
	@echo "〉go generate"
	@go generate ./...

### Dependencies

.PHONY: audit
## Run security audit
audit:
	@echo "〉security audit"
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...

.PHONY: outdated
## Show outdated direct dependencies
outdated:
	@echo "〉go mod outdated"
	@go-mod-upgrade --list

.PHONY: upgrade
## Show outdated direct dependencies
upgrade:
	@echo "〉go mod upgrade"
	@go-mod-upgrade
	@$(MAKE) tidy

### Documentation

.PHONY: docs
## Open docs
docs:
	@echo "〉starting docs"
	@cd docs && bun install && bun run dev

.PHONY: docs.build
## Open docs
docs.build:
	@echo "〉building docs"
	@cd docs && bun install && bun run build

.PHONY: godocs
## Open go docs
godocs:
	@echo "〉starting go docs"
	@go doc -http

### Utils

.PHONY: help
# https://patorjk.com/software/taag/#p=display&f=Tmplr&t=gotsrpc&x=none&v=4&h=4&w=80&we=false
## Show help text
help: g=\033[0;32m
help: b=\033[0;34m
help: w=\033[0;90m
help: e=\033[0m
help:
	@echo "$(g)"
	@echo ""
	@echo "    ┏"
	@echo "┏┓┏┓╋┓┏┏┓┏┓┏"
	@echo "┗┫┗┛┛┗┻┛┗┗┗┫"
	@echo " ┛         ┛"
	@echo "with ❤ foomo by bestbytes"
	@echo "$(e)"
	@echo "$(b)Usage:$(e)\n  make [task]"
	@awk '{ \
		if($$0 ~ /^### /){ \
			if(help) printf "  %-21s $(w)%s$(e)\n\n", cmd, help; help=""; \
			printf "$(b)\n%s:$(e)\n", substr($$0,5); \
		} else if($$0 ~ /^[a-zA-Z0-9._-]+:/){ \
			cmd = substr($$0, 1, index($$0, ":")-1); \
			if(help) printf "  %-21s $(w)%s$(e)\n", cmd, help; help=""; \
		} else if($$0 ~ /^##/){ \
			help = help ? help "\n                        " substr($$0,3) : substr($$0,3); \
		} else if(help){ \
			print "\n                        $(w)" help "$(e)\n"; help=""; \
		} \
	}' $(MAKEFILE_LIST)
	@echo ""

