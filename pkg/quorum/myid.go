package quorum

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"strconv"
	"sync"
)

// Models the myid file used by Zookeeper.  For some reason, Exhibitor
// sometimes will delete this file and this causes problems. This will
// detect any chances and ensure it's always there.
type MyIdFile struct {
	Path  string
	Value int

	Error <-chan error

	stop chan<- interface{}
	lock sync.Mutex
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

func (this *MyIdFile) Close() error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.stop != nil {
		this.stop <- 1
	}
	return nil
}

// Continuously watching the path and ensures that the file matches the state of the id.
func (this *MyIdFile) EnsureState() error {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.stop != nil {
		// already running.
		return nil
	}

	stop := make(chan interface{})
	error := make(chan error)

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
					log.Warn("myid at", this.Path, "removed.  Recreating.")
					err := this.Create()
					if err != nil {
						log.Warn("Cannot create file")
						error <- err
						break
					}
				case fsnotify.Write:
				default:
				}
			case err := <-watcher.Errors:
				log.Warn("Error:", err)
				error <- err
			case <-stop:
				break
			}
		}
		log.Info("Stopped")
	}()

	this.Error = error
	this.stop = stop
	return nil

}
