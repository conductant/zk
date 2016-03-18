package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
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
	command.RegisterFunc("setup", config,
		func(a []string, w io.Writer) error {
			defer config.Close()

			if err := config.Init(); err != nil {
				return err
			}
			log.Info("Initialized")

			buff, err := config.GenerateConfig()
			if err != nil {
				return err
			}
			log.Info("Generated config:", string(buff))

			err = config.Exhibitor.Start()
			if err != nil {
				return err
			}

			// Block until Exhibitor is up
			<-config.Exhibitor.Ready

			log.Info("Applying config")

			config.Exhibitor.ApplyConfig("", buff)

			// TODO - loop here to check the status of zookeeper.

			log.Info("Zookeeper ready.")

			// Block forever....
			done := make(chan bool)
			<-done

			return nil
		},
		func(w io.Writer) {
			fmt.Fprintln(w, "Bootstraps an ensemble member")
		})

	command.RegisterFunc("print-config", config,
		func(a []string, w io.Writer) error {
			defer config.Close()

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
		func(w io.Writer) {
			fmt.Fprintln(w, "For test only")
		})

	runtime.Main()
}
