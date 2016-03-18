Zookeeper
=========

This Zookeeper container contains a bootstrapper, Apache Zookeeper, and Netflix's Exhibitor.  Management of Zookeeper
ensemble is done via the embedded Exhibitor.  The bootstrapper starts Exhibitor and uses it for configuration of
the ensemble and then waits for Zookeeper ensemble to be online and ready.

## Usage

    docker run --rm -ti conductant/zk:latest -h

## Bootstrap an ensemble

To start the ensemble, for each node, start the container with a list of IP addresses for the nodes that are
in the quorum and those that are observers.  The bootstrapper will automatically configure the ensemble.  For example,
let's say we have the following 3 quorum members and 2 observers:

```
    host1 192.168.99.100 - Quorum member (S)
    host2 192.168.99.101 - Quorum member (S)
    host3 192.168.99.102 - Quorum member (S)
    host4 192.168.99.103 - Observer (O)
    host5 192.168.99.104 - Observer (O)
```
On each host, start up the container using the `bootstrap` command and flags to indicate the members of the ensemble.
Using Docker Machine to point the local docker client to different hosts, then command looks like:

```
    docker $(docker-machine config host1) run -d --name zk \
        -p 8080:8080 -p 2888:2888 -p 3888:3888 -p 2181:2181 \
        conductant/zk:latest bootstrap \
	-ip 192.168.99.100 \
	-S 192.168.99.100 \
	-S 192.168.99.101 \
	-S 192.168.99.102 \
	-O 192.168.99.103 \
	-O 192.168.99.104 \
```

## Using Docker Machine

Docker-Machine can be used to manage this.  The file `etc/demo-cluster.mk` shows how.  To spin up a new ZK cluster
locally using the `virtualbox` driver, do this from the top level directory:

```
% make etc/demo-cluster.mk
```

This will provision a list of hosts `zk-1`, `zk-2`, `zk-3`.  Using Docker Machine

```
$ docker-machine ls
NAME      ACTIVE   DRIVER       STATE     URL                         SWARM   DOCKER    ERRORS
default   -        virtualbox   Running   tcp://192.168.99.161:2376           v1.10.2   
zk-1      -        virtualbox   Running   tcp://192.168.99.181:2376           v1.10.3   
zk-2      -        virtualbox   Running   tcp://192.168.99.182:2376           v1.10.3   
zk-3      -        virtualbox   Running   tcp://192.168.99.183:2376           v1.10.3
```

You can then go to any one of the`zk-` hosts to view the Exhibitor UI.  For example:

```
http://192.168.99.181:8080/exhibitor/v1/ui/index.html
```

## Print configuration

To get the configuration, use the `print-config` command:

```
    docker $(docker-machine config host1) run --rm -ti \
        conductant/zk:latest print-config \
	-ip 192.168.99.100 \
	-S 192.168.99.100 \
	-S 192.168.99.101 \
	-S 192.168.99.102 \
	-O 192.168.99.103 \
	-O 192.168.99.104 \
```

This would print out for example:

```
INFO[0000] MyID file ready{
    "config": {
      "autoManageInstances": "0",
      "autoManageInstancesApplyAllAtOnce": "1",
      "autoManageInstancesFixedEnsembleSize": "0",
      "autoManageInstancesSettlingPeriodMs": "180000",
      "backupExtra": {},
      "backupMaxStoreMs": "86400000",
      "backupPeriodMs": "60000",
      "checkMs": "30000",
      "cleanupMaxFiles": "3",
      "cleanupPeriodMs": "43200000",
      "clientPort": "2181",
      "connectPort": "2888",
      "electionPort": "3888",
      "javaEnvironment": "",
      "log4jProperties": "",
      "logIndexDirectory": "",
      "observerThreshold": "999",
      "serverId": 1,
      "serversSpec": "S:1:0.0.0.0,S:2:192.168.99.101,S:3:192.168.99.102,O:4:192.168.99.103,O:5:192.168.99.104",
      "zooCfgExtra": {
        "initLimit": "10",
        "syncLimit": "5",
        "tickTime": "2000"
      },
      "zookeeperDataDirectory": "/var/zookeeper",
      "zookeeperInstallDirectory": "/usr/local/zookeeper",
      "zookeeperLogDirectory": ""
    },
    "myid": 1,
    "zk_hosts": "192.168.99.103:2181,192.168.99.104:2181,192.168.99.100:2181,192.168.99.101:2181,192.168.99.102:2181"
  }
```