package quorum

const (
	DefaultZkExhibitorConfigTemplate = `{
    "zookeeperInstallDirectory":"/usr/local/zookeeper",
    "zookeeperDataDirectory":"/var/zookeeper",
    "zookeeperLogDirectory":"",
    "logIndexDirectory":"",
    "autoManageInstancesSettlingPeriodMs":"180000",
    "autoManageInstancesFixedEnsembleSize":"0",
    "autoManageInstancesApplyAllAtOnce":"1",
    "observerThreshold":"999",
    "serversSpec":"{{ zk_servers_spec }}",
    "javaEnvironment":"",
    "log4jProperties":"",
    "clientPort":"2181",
    "connectPort":"2888",
    "electionPort":"3888",
    "checkMs":"30000",
    "cleanupPeriodMs":"43200000",
    "cleanupMaxFiles":"3",
    "backupPeriodMs":"60000",
    "backupMaxStoreMs":"86400000",
    "autoManageInstances":"0",
    "zooCfgExtra":{
        "syncLimit":"5",
	"tickTime":"2000",
	"initLimit":"10"
    },
    "backupExtra":{},
    "serverId":{{server_id}}
}
`
)
