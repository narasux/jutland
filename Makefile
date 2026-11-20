.PHONY: tidy build pack test fmt vet sim-ci sim-periodic sim-watch sim-archive sim-frame sim-frame-gate sim-perf sim-suite sim-nightly sim-schedule-install sim-schedule-uninstall sim-schedule-status golines gofumpt turret-marker

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

# local simulation CI: unit tests + vet + build
sim-ci:
	bash tools/sim-ci.sh

# periodic trend/analysis from the private lab history
sim-periodic:
	bash tools/sim-periodic.sh

# periodic trend/analysis until interrupted
sim-watch:
	go run ./cmd/jutland-sim watch --interval "$${SIM_WATCH_INTERVAL:-10m}"

# move old runs out of reports/runs (dry-run first)
sim-archive:
	go run ./cmd/jutland-sim archive --keep "$${SIM_KEEP_RUNS:-50}" --dry-run

# rendered-frame p99 (opens a temporary window; not for headless CI)
sim-frame:
	go run ./cmd/jutland-frame --frames "$${SIM_FRAME_COUNT:-300}" --budget-ms "$${SIM_FRAME_BUDGET_MS:-16.7}"

# rendered-frame p99 gate (requires GUI)
sim-frame-gate:
	JUTLAND_SIM_HOME="$${JUTLAND_SIM_HOME:?}" ./tools/frame-gate.sh

# p99 gate over the newest F3 run
sim-perf:
	bash tools/perf-gate.sh

# run a small F0/F1/F3 suite, then trend/analyze
sim-suite:
	bash tools/sim-suite.sh

# nightly lab run: suite + rendered-frame gate + trend/analyze
sim-nightly:
	bash tools/sim-nightly.sh

# install/uninstall/status the nightly launchd job (macOS, explicit only)
sim-schedule-install:
	bash tools/install-scheduler.sh --install

sim-schedule-uninstall:
	bash tools/install-scheduler.sh --uninstall

sim-schedule-status:
	bash tools/install-scheduler.sh --status

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
	python3 utils/turret_marker/server.py "$(IMAGE)" $(if $(ROTATE),--rotate $(ROTATE),) $(if $(TYPES),--types '$(TYPES)',)
