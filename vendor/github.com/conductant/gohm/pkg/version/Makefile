all: test-version

# Common makefile that extracts git version info and generates the LDFLAGS variable.
include ../../hack/make/version.mk

test-version:
	${GODEP} go test -ldflags "$(LDFLAGS)" ./...  -logtostderr -check.vv -v ${TEST_ARGS}

build-version:
	${GODEP} go build -ldflags "$(LDFLAGS)" ./...
