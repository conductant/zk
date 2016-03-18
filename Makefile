PKGS=$(wildcard pkg/*)
clean_PKGS=$(addprefix clean_,$(PKGS))

all: $(PKGS)
clean: $(clean_PKGS)

.PHONY: force
$(PKGS): force
	make -C $@

$(clean_PKGS): force
	make -C $(patsubst clean_%,%,$@) clean

bin:
	cd cmd && make clean build-zk-linux

GIT_TAG=`git describe --abbrev=0 --tags`
BUILD_DOCKER_IMAGE?=conductant/zk:$(GIT_TAG)

docker-dev: bin
	docker build -t $(BUILD_DOCKER_IMAGE) -f Dockerfile.dev .
	docker tag $(BUILD_DOCKER_IMAGE) conductant/zk:latest

docker: bin
	docker build -t $(BUILD_DOCKER_IMAGE) .
	docker tag $(BUILD_DOCKER_IMAGE) conductant/zk:latest

push: docker
	docker push $(BUILD_DOCKER_IMAGE)
	docker push conductant/zk:latest
