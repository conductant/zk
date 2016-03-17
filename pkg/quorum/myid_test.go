package quorum

import (
	. "gopkg.in/check.v1"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMyIdFile(t *testing.T) { TestingT(t) }

type TestSuiteMyIdFile struct {
}

var _ = Suite(&TestSuiteMyIdFile{})

func (suite *TestSuiteMyIdFile) SetUpSuite(c *C) {
}

func (suite *TestSuiteMyIdFile) TearDownSuite(c *C) {
}

func (suite *TestSuiteMyIdFile) TestMyIdFile(c *C) {
	myid := &MyIdFile{
		Path:  filepath.Join(c.MkDir(), "myid"),
		Value: 1,
	}

	c.Assert(myid.Exists(), Equals, false)

	err := myid.Create()
	c.Assert(err, IsNil)

	c.Assert(myid.Exists(), Equals, true)
}

func (suite *TestSuiteMyIdFile) TestEnsureState(c *C) {
	myid := &MyIdFile{
		Path:  filepath.Join(c.MkDir(), "myid"),
		Value: 1,
	}

	stop := make(chan interface{})
	error := make(chan error)

	err := myid.EnsureState(stop, error)
	c.Assert(err, IsNil)

	// remove it
	err = os.Remove(myid.Path)
	c.Assert(err, IsNil)

	time.Sleep(1 * time.Second)

	c.Assert(myid.Exists(), Equals, true)

	close(stop)
}
