package main

import (
	"github.com/conductant/gohm/pkg/command"
	"github.com/conductant/gohm/pkg/runtime"
	"github.com/conductant/zk/pkg/quorum"
)

func main() {

	command.Register("setup",
		func() (command.Module, command.ErrorHandling) {
			return &quorum.Config{
				Servers: []quorum.HostPort{
					quorum.HostPort("default:1234"),
				},
			}, command.PanicOnError
		})

	runtime.Main()
}
