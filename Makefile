# disable default rules
.SUFFIXES:
MAKEFLAGS+=-r -R
default: test

.PHONY: test
test:
	python3 tools/check_spec.py
	go test ./...
	go test -race -shuffle=on ./...

.PHONY: acceptance
acceptance:
	python3 tools/check_acceptance.py

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: ci-tidy
ci-tidy:
	go mod tidy
	@test -z "$$(git status --porcelain -- go.mod go.sum)" || { echo "Please run 'go mod tidy'."; exit 1; }

.PHONY: staticcheck
staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
