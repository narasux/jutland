.PHONY: tidy build pack test fmt vet golines gofumpt turret-marker

ifdef VERSION
    VERSION=${VERSION}
else
    VERSION=$(shell git describe --always)
endif

GOARCH ?= arm64
GOOS ?= darwin
BINARY_NAME ?= jutland

GITCOMMIT=$(shell git rev-parse HEAD)
BUILDTIME=${shell date +%Y-%m-%dT%I:%M:%S}

LDFLAGS="-X github.com/narasux/jutland/pkg/version.Version=${VERSION} \
	-X github.com/narasux/jutland/pkg/version.GitCommit=${GITCOMMIT} \
	-X github.com/narasux/jutland/pkg/version.BuildTime=${BUILDTIME}"

# go mod tidy
tidy:
	go mod tidy

# build executable binary
build: tidy
	go build -ldflags ${LDFLAGS} -o jutland ./main.go

# run unittest
test: tidy
	go test ./...

# build game package
pack:
	# clean pack dir
	rm -rf ./jutland-${VERSION}

	# init pack dir
	mkdir ./jutland-${VERSION}

	# copy Readme.md
	# TODO Readme.pdf maybe better?
	cp ./Readme.md ./jutland-${VERSION}/Readme.md

	# copy resource files
	cp -r ./resources ./jutland-${VERSION}/resources

	# copy configs files
	cp -r ./configs ./jutland-${VERSION}/configs

	# build executable binary
	GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags ${LDFLAGS} -o ./jutland-${VERSION}/${BINARY_NAME} ./main.go

	# build zip package
	zip -r jutland-${GOOS}-${GOARCH}-${VERSION}.zip jutland-${VERSION}

	# clean pack dir
	rm -rf ./jutland-${VERSION}

fmt: golines gofumpt
	$(GOLINES) ./ -m 119 -w --base-formatter gofmt --no-reformat-tags
	$(GOFUMPT) -l -w .

vet:
	go vet ./...

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLINES ?= $(LOCALBIN)/golines
GOFUMPT ?= $(LOCALBIN)/gofumpt

golines: $(GOLINES)
$(GOLINES): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/segmentio/golines@v0.13.0

gofumpt: $(GOFUMPT)
$(GOFUMPT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install mvdan.cc/gofumpt@v0.10.0

# launch the browser-based ship turret position marker
turret-marker:
	@test -n "$(IMAGE)" || (echo "Usage: make turret-marker IMAGE=path/to/top.png" && exit 1)
	python3 utils/turret_marker/server.py "$(IMAGE)" $(if $(ROTATE),--rotate $(ROTATE),)
