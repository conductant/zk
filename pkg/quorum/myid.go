package quorum

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"io/ioutil"
	"strconv"
)

// Models the myid file used by Zookeeper.  For some reason, Exhibitor
// sometimes will delete this file and this causes problems. This will
// detect any chances and ensure it's always there.
type MyIdFile struct {
	Path  string
	Value int
}

// Reads the file at the path and compares the read value with self.
func (this *MyIdFile) Exists() bool {
	bytes, err := ioutil.ReadFile(this.Path)
	if err != nil {
		return false
	}
	v, err := strconv.Atoi(string(bytes))
	if err != nil {
		return false
	}
	return v == this.Value
}

func (this *MyIdFile) Create() error {
	return ioutil.WriteFile(this.Path, []byte(fmt.Sprintf("%d", this.Value)), 0666)
}

// Continuously watching the path and ensures that the file matches the state of the id.
func (this *MyIdFile) EnsureState(stop <-chan interface{}, error chan<- error) error {
	if !this.Exists() {
		if err := this.Create(); err != nil {
			return err
		}
	}

	// watch this file now
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcher.Add(this.Path)
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				switch event.Op {
				case fsnotify.Remove, fsnotify.Rename:
					glog.Warningln("myid at", this.Path, "removed.  Recreating.")
					err := this.Create()
					if err != nil {
						glog.Warningln("Cannot create file")
						error <- err
						break
					}
				case fsnotify.Write:
				default:
				}
			case err := <-watcher.Errors:
				glog.Warningln("Error:", err)
				error <- err
			case <-stop:
				break
			}
		}
		glog.Infoln("Stopped")
	}()
	return nil

}
