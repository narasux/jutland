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

# format code style
fmt:
	golines ./ -m 120 -w --base-formatter gofmt --no-reformat-tags
	gofumpt -l -w .

# run unittest
test: tidy
	go test ./...

# build game package
pack:
	# clean pack dir
	rm -rf ./jutland-${VERSION}

	# init pack dir
	mkdir ./jutland-${VERSION}

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
