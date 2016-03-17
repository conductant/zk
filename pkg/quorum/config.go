package quorum

import (
	"errors"
	"fmt"
	"github.com/conductant/gohm/pkg/conf"
	"github.com/golang/glog"
	"io"
	"strings"
)

const (
	MyIdFilePath = "/var/zookeeper/myid"
)

type HostPort string

type Config struct {
	conf.Conf `json:"-" yaml:"-"`
	Exhibitor

	Servers   []HostPort `json:"servers" yaml:"servers" flag:"S, Quorum members of <host>:<port>"`
	Observers []HostPort `json:"observers" yaml:"observers" flag:"O, Quorum observers of <host>:<port>"`

	Hostname string `flag:"ip, This host's name or ip address"`
	MyIdPath string `flag:"myid_path, MyId location"`

	self     *Server
	ensemble []*Server // de-duped, sorted
}

type Server struct {
	Ip       string
	Port     int
	Observer bool
}

func (this *Config) Help(w io.Writer) {
	fmt.Fprintln(w, "Configure the quorum")
}

func (this *Config) Run(args []string, w io.Writer) error {
	if err := this.Init(); err != nil {
		return err
	}
	glog.Infoln("Initialized")

	// MyId
	myid := &MyIdFile{
		Path:  this.MyIdPath,
		Value: this.GetMyId(),
	}

	stopChan := make(chan interface{})
	errChan := make(chan error)
	if err := myid.EnsureState(stopChan, errChan); err != nil {
		return err
	}
	glog.Infoln("MyID file ready")

	config, err := this.GenerateConfig()
	if err != nil {
		return err
	}
	glog.Infoln("Generated config:", string(config))

	this.Exhibitor.CheckZkReady()

	this.Exhibitor.ApplyConfig("", config)

	<-this.Exhibitor.ZkReady
	glog.Infoln("Zookeeper ready.")

	// Block forever....
	done := make(chan bool)
	<-done

	stopChan <- 1
	return nil
}

func (this *Config) Close() error {
	return nil
}

func (this *Config) Init() error {
	all := map[string]*Server{}
	for _, hp := range this.Observers {
		if s, err := hp.toServer(); err == nil {
			s.Observer = true
			all[s.Ip] = s
		} else {
			return err
		}
	}
	for _, hp := range this.Servers {
		if s, err := hp.toServer(); err == nil {
			all[s.Ip] = s
		} else {
			return err
		}
	}

	sorter := new(serverSorter)
	for _, s := range all {
		sorter.Add(s)
	}
	sorter.Sort()
	this.ensemble = sorter.servers
	this.self = &Server{Ip: this.Hostname}
	return nil
}

func (this *Config) GenerateConfig() ([]byte, error) {
	return this.Exhibitor.GenerateConfig(this, this.templateFuncs())
}

func (this *Config) templateFuncs() map[string]interface{} {
	return map[string]interface{}{
		"zk_hosts": func() string {
			return this.GetZkHosts()
		},
		"zk_servers_spec": func() string {
			return this.GetZkServersSpec()
		},
		"zk_default_template": func() string {
			return DefaultZkExhibitorConfigTemplate
		},
		"server_id": func() string {
			return fmt.Sprintf("%d", this.GetMyId())
		},
	}
}

func (this *Config) GetMyId() int {
	for id, s := range this.ensemble {
		if this.self.Ip == s.Ip {
			return id + 1
		}
	}
	panic(errors.New("err-cannot-determine-myid"))
}

// Generates the quorum server list
func (this *Config) GetZkServersSpec() string {
	list := []string{}
	for id, s := range this.ensemble {
		serverType := "S"
		if s.Observer {
			serverType = "O"
		}
		host := s.Ip
		if this.self.Ip == s.Ip {
			host = "0.0.0.0"
		}
		list = append(list, fmt.Sprintf("%s:%d:%s", serverType, id+1, host))
	}
	return strings.Join(list, ",")
}

// Generates the client connection hosts string
func (this *Config) GetZkHosts() string {
	list := []string{}
	// Get from observers if any
	for _, s := range this.ensemble {
		if s.Observer {
			host := s.Ip
			port := 2181
			if s.Port > 0 {
				port = s.Port
			}
			list = append(list, fmt.Sprintf("%s:%d", host, port))
		}
	}
	// If nothing, then just connect to servers
	if len(list) < 3 {
		for _, s := range this.ensemble {
			if !s.Observer {
				host := s.Ip
				port := 2181
				if s.Port > 0 {
					port = s.Port
				}
				list = append(list, fmt.Sprintf("%s:%d", host, port))
			}
		}
	}
	return strings.Join(list, ",")
}
