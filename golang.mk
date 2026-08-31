TARGET?="./cmd/gotpl-linter"
PACKAGE=$(shell GOPATH= go list $(TARGET))
NAME=$(shell echo $(PACKAGE) | awk -F "/" '{print $$NF}')

BUILD_VERSION=$(shell git describe --always --dirty --tags | tr '-' '.' )
BUILD_DATE=$(shell LC_ALL=C date)
BUILD_HASH=$(shell git rev-parse HEAD)

#BUILD_XDST=$(NAME)
BUILD_XDST=main
BUILD_FLAGS=-ldflags "\
	$(ADDITIONAL_LDFLAGS) -s -w \
	-X '$(BUILD_XDST).BuildVersion=$(BUILD_VERSION)' \
	-X '$(BUILD_XDST).BuildDate=$(BUILD_DATE)' \
	-X '$(BUILD_XDST).BuildHash=$(BUILD_HASH)' \
"

GOFILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.git/*")
GOPKGS=$(shell go list ./cmd/...)

OUTPUT_FILE=$(NAME)-$(BUILD_VERSION)-$(shell go env GOOS)-$(shell go env GOARCH)$(shell go env GOARM)$(shell go env GOEXE)
OUTPUT_LINK=$(NAME)$(shell go env GOEXE)
WINDOWS_ZIP=$(shell echo $(OUTPUT_FILE) | sed 's/\.exe/\.zip/')

default: build

vendor: go.mod go.sum
	go mod vendor
	touch vendor

format:
	gofmt -s -w $(GOFILES)

vet: vendor
	go vet $(GOPKGS)

lint:
	$(foreach pkg,$(GOPKGS),golint $(pkg);)

test_packages: vendor
	go test $(GOPKGS)

test_format:
	gofmt -s -l $(GOFILES)

test: test_format vet lint test_packages

cov:
	gocov test -v $(GOPKGS) \
		| gocov-html > coverage.html

_build: vendor
	mkdir -p dist
	go build $(BUILD_FLAGS) -o dist/$(OUTPUT_FILE) $(TARGET);

build: _build
	$(foreach TARGET,$(TARGETS),ln -sf $(OUTPUT_FILE) dist/$(OUTPUT_LINK);)

compress: _build
	tar czf dist/$(OUTPUT_FILE).tar.gz -C dist $(OUTPUT_FILE)
	rm -f dist/$(OUTPUT_FILE)

xc:
	GOOS=linux GOARCH=amd64 make compress
	GOOS=linux GOARCH=arm64 make compress
	GOOS=linux GOARCH=arm GOARM=7 make compress
	GOOS=darwin GOARCH=amd64 make compress
	GOOS=darwin GOARCH=arm64 make compress
	GOOS=windows GOARCH=amd64 make compress

xb:
	GOOS=linux GOARCH=amd64 make build
	GOOS=linux GOARCH=arm64 make build
	GOOS=linux GOARCH=arm GOARM=7 make build
	GOOS=darwin GOARCH=amd64 make build
	GOOS=darwin GOARCH=arm64 make build
	GOOS=windows GOARCH=amd64 make build

clean:
	rm dist/ -rvf
	rm mocks/ -rvf

IMAGE ?= ghcr.io/imunhatep/gotpl-yaml-linter
PLATFORMS := linux/amd64,linux/arm64

# Build a single-arch image for the local host.
.PHONY: image
image:
	podman build \
		--build-arg VERSION=$(BUILD_VERSION) \
		--build-arg COMMIT=$(BUILD_HASH) \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		-f Containerfile \
		-t $(IMAGE):$(BUILD_VERSION) .

# Build a multi-arch manifest. Override IMAGE=registry/name to tag elsewhere, e.g.:
#   make image-multiarch IMAGE=ghcr.io/someone/gotpl-yaml-linter
.PHONY: image-multiarch
image-multiarch:
	podman build --platform $(PLATFORMS) --manifest $(IMAGE):$(BUILD_VERSION) \
		--build-arg VERSION=$(BUILD_VERSION) \
		--build-arg COMMIT=$(BUILD_HASH) \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		-f Containerfile .

.PHONY: docker-test
docker-test:
	$(DOCKER) build -t gotpl_linter:test -f build/test/Dockerfile .

.PHONY: update-deps
update-deps:
	@echo ">> updating Go dependencies"
	@for m in $$(go list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all); do \
		go get $$m; \
	done
	go mod tidy
ifneq (,$(wildcard vendor))
	go mod vendor
endif