.PHONY: tidy build test

ifdef VERSION
    VERSION=${VERSION}
else
    VERSION=$(shell git describe --always)
endif

GOARCH ?= amd64
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
	# init pack dir
	mkdir ./jutland-${VERSION}

	# copy resource files
	cp -r ./resources ./jutland-${VERSION}/resources

	# build executable binary
	GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags ${LDFLAGS} -o ./jutland-${VERSION}/${BINARY_NAME} ./main.go

	# build zip package
	zip -r jutland-${GOOS}-${GOARCH}-${VERSION}.zip jutland-${VERSION}

	# clean pack dir
	rm -rf ./jutland-${VERSION}
