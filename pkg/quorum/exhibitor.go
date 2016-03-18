package quorum

import (
	"bytes"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/conductant/gohm/pkg/encoding"
	"github.com/conductant/gohm/pkg/resource"
	"github.com/conductant/gohm/pkg/template"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	ZkLocalExhibitorConfigEndpoint      = "http://localhost:8080/exhibitor/v1/config/set"
	ZkLocalExhibitorCheckStatusEndpoint = "http://localhost:8080/exhibitor/v1/config/get-state"
	ZkLocalExhibitorStartCommand        = "java -jar /usr/local/exhibitor-1.5.1/exhibitor-1.5.1.jar -c file"
)

type Exhibitor struct {
	ZkReadyTimeout      encoding.Duration  `json:"zk_ready_timeout" yaml:"zk_ready_timeout"`
	ZkReadyPollInterval encoding.Duration  `json:"zk_ready_poll_interval" yaml:"zk_ready_poll_interval"`
	ConfigTemplateUrl   string             `json:"config_url" yaml:"config_url" flag:"t, Url of config template."`
	ConfigEndpoint      string             `json:"config_endpoint" yaml:"config_endpoint"`
	CheckStatusEndpoint string             `json:"status_endpoint" yaml:"status_endpoint"`
	Ready               <-chan interface{} `json:"-" yaml:"-"`
	Error               <-chan error       `json:"-" yaml:"-"`

	cmd *exec.Cmd
}

func (this *Exhibitor) Start() error {
	err := this.exec()
	if err != nil {
		return err
	}
	this.checkReady()
	return nil
}

func (this *Exhibitor) Stop() error {
	if this.cmd == nil {
		return errors.New("err-not-running")
	}
	err := this.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}
	state, err := this.cmd.Process.Wait()
	log.Info("Process exited=", state.Exited(), "state=", state.String())
	return err
}

func (this *Exhibitor) exec() error {
	command := strings.Split(ZkLocalExhibitorStartCommand, " ")
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	this.cmd = cmd
	return nil
}

func (this *Exhibitor) checkReady() {
	ready := make(chan interface{})
	error := make(chan error, 100)

	this.Ready = ready
	this.Error = error

	go func() {

		ticker := time.Tick(this.ZkReadyPollInterval.Duration)
		timeout := time.Tick(this.ZkReadyTimeout.Duration)
		for {
			select {

			case <-timeout:
				error <- errors.New("err-timeout-zk-startup")

			case <-ticker:

				log.Info("CheckReady: ", this.CheckStatusEndpoint)

				client := &http.Client{}
				resp, err := client.Get(this.CheckStatusEndpoint)

				log.Info("CheckReady resp=", resp, "Err=", err)

				if err == nil && resp.StatusCode == http.StatusOK {

					buff, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						error <- err
					}

					status := new(struct {
						Running bool `json:"running"`
					})
					err = json.Unmarshal(buff, status)
					log.Info("Status=", string(buff), "err=", err)

					// At this point, ready or not just as long we have a response
					if err == nil {
						log.Info("Got valid response from Exhibitor: server running=", status.Running)
						if status.Running {
							close(ready) // no longer blocks
						}
						return
					} else {
						log.Info("Exhibitor not running. Wait.")
					}
				}
			}
		}
	}()
}

func (this *Exhibitor) GenerateConfig(data interface{}, funcs map[string]interface{}) ([]byte, error) {
	tpl, err := resource.Fetch(context.Background(), this.ConfigTemplateUrl)
	if err != nil {
		tpl = []byte(DefaultZkExhibitorConfigTemplate)
	}
	return template.Apply(tpl, data, funcs)
}

func (this *Exhibitor) ApplyConfig(authToken string, config []byte) error {
	// now apply the config, based on the url of the destination
	parts := strings.Split(this.ConfigEndpoint, "://")
	if len(parts) == 1 {
		return errors.New("err-bad-url:" + this.ConfigEndpoint)
	}
	switch parts[0] {
	case "http", "https":
		return do_post(this.ConfigEndpoint, config, authToken)
	case "file":
		return do_save(parts[1], config)
	default:
		return errors.New("err-not-supported:" + parts[0])
	}
	return nil
}

func do_post(url string, body []byte, authToken string) error {
	client := &http.Client{}
	post, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	post.Header.Add("Content-Type", "application/json")
	if authToken != "" {
		post.Header.Add("Authorization", "Bearer "+authToken)
	}
	resp, err := client.Do(post)
	if err != nil {
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return errors.New("err-post-failed:" + url)
	}
}

func do_save(path string, body []byte) error {
	return ioutil.WriteFile(path, []byte(body), 0777)
}
