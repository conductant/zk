# Demo makefile for building a new zookeeper cluster
# This assumes all nodes are voting members of the quorum.
# Observers are possible with the -O option but isn't included here.
# Also this uses virtualbox as Docker Machine driver.

NODES?= zk-1 zk-2 zk-3
LABEL?=cluster=zookeeper

all: rm-zk-nodes create-zk-nodes start-zk
clean: rm-zk-nodes


rm-zk-nodes:
	-$(foreach i,$(NODES), \
	docker-machine rm -f $(i); \
	)
	echo Docker machines:
	docker-machine ls

create-zk-nodes: rm-zk-nodes
	-$(foreach i,$(NODES), \
	docker-machine create -d virtualbox \
	--engine-opt="label=$(LABEL)" \
	$(i);\
	)
	echo Docker machines:
	docker-machine ls

ps-all:
	-$(foreach i,$(NODES), \
	docker `docker-machine config $(i)` ps -a ; \
	)

ensemble:
	-echo $(foreach i, $(NODES), -S `docker-machine ip $(i)`)

SERVERS=$(foreach i, $(NODES), -S `docker-machine ip $(i)`)
start-zk:
	$(foreach i,$(NODES), \
	docker `docker-machine config $(i)` run -d --name zk.$(i) \
		-p 8080:8080 -p 2181:2181 -p 2888:2888 -p 3888:3888 \
		conductant/zk:latest bootstrap -ip `docker-machine ip $(i)` \
		$(SERVERS) ; \
	)

