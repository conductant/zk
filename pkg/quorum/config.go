package quorum

import (
	"fmt"
	"github.com/conductant/gohm/pkg/conf"
	"github.com/golang/glog"
	"io"
)

type HostPort string

type Config struct {
	conf.Conf `json:"-" yaml:"-"`

	Servers   []HostPort `json:"servers" yaml:"servers" flag:"S, Quorum members of <host>:<port>"`
	Observers []HostPort `json:"observers" yaml:"observers" flag:"O, Quorum observers of <host>:<port>"`
}

func (this *Config) Help(w io.Writer) {
	fmt.Fprintln(w, "Configure the quorum")
}

func (this *Config) Run(args []string, w io.Writer) error {
	glog.Infoln("Running with", this)
	return nil
}

func (this *Config) Close() error {
	return nil
}
