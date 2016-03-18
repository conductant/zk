all: clean build-zk

clean:
	-rm zk
	-rm -rf ../build
	mkdir -p ../build

# Common makefile that extracts git version info and generates the LDFLAGS variable.
include ../hack/make/version.mk

build-zk: clean build-zk-osx build-zk-linux

build-zk-osx:
	GOOS=darwin GOARCH=amd64 \
	${GODEP} go build -v -ldflags "$(LDFLAGS)" -o ../build/darwin-amd64/zk zk.go

build-zk-linux:
	GOOS=linux GOARCH=amd64 \
	${GODEP} go build -v -ldflags "$(LDFLAGS)" -o ../build/linux-amd64/zk zk.go
