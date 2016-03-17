package main

import (
	"encoding/json"
	"fmt"
	"github.com/conductant/gohm/pkg/command"
	"github.com/conductant/gohm/pkg/runtime"
	"github.com/conductant/zk/pkg/quorum"
	"io"
)

func main() {

	config := &quorum.Config{
		MyIdPath: quorum.MyIdFilePath,
		Exhibitor: quorum.Exhibitor{
			ConfigEndpoint:      quorum.ZkLocalExhibitorConfigEndpoint,
			CheckStatusEndpoint: quorum.ZkLocalExhibitorCheckStatusEndpoint,
		},
	}
	command.Register("setup",
		func() (command.Module, command.ErrorHandling) {
			return config, command.PanicOnError
		})
	command.RegisterFunc("print-config", config,
		func(a []string, w io.Writer) error {
			config.Init()
			buff, err := config.GenerateConfig()
			if err != nil {
				panic(err)
			}
			c := map[string]interface{}{}
			err = json.Unmarshal(buff, &c)
			if err != nil {
				return err
			}

			// Dump out in json format
			m := map[string]interface{}{
				"myid":     config.GetMyId(),
				"zk_hosts": config.GetZkHosts(),
				"config":   c,
			}
			buff, err = json.MarshalIndent(m, "  ", "  ")
			if err != nil {
				return err
			}
			fmt.Fprint(w, string(buff))
			return nil
		},
		func(w io.Writer) { fmt.Fprintln(w, "For test only") })

	runtime.Main()
}
