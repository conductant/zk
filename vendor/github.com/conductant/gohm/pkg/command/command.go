package command

import (
	"flag"
	"fmt"
	gflag "github.com/conductant/gohm/pkg/flag"
	"io"
	"os"
	"reflect"
	"sort"
	"sync"
)

type ErrorHandling flag.ErrorHandling

const (
	ContinueOnError = ErrorHandling(flag.ContinueOnError)
	PanicOnError    = ErrorHandling(flag.PanicOnError)
	ExitOnError     = ErrorHandling(flag.ExitOnError)
)

var (
	lock     sync.Mutex
	modules  = map[string]func() (Module, ErrorHandling){}
	policies = map[string]flag.ErrorHandling{}

	reparseLock  sync.Mutex
	reparseFlags = map[reflect.Type]func(){}
)

func Register(module string, commandFunc func() (Module, ErrorHandling)) {
	lock.Lock()
	defer lock.Unlock()
	modules[module] = commandFunc
	policies[module] = flag.PanicOnError // default
}

func RegisterFunc(module string, obj interface{}, run func([]string, io.Writer) error, help ...func(io.Writer)) {
	lock.Lock()
	defer lock.Unlock()
	h := func(w io.Writer) {
		fmt.Fprintln(w, "no help provided")
	}
	if len(help) > 0 {
		h = help[0]
	}

	modules[module] = func() (Module, ErrorHandling) {
		return &module_adapter{obj: obj, f: run, h: h}, PanicOnError
	}
	policies[module] = flag.PanicOnError // default
}

type module_adapter struct {
	obj interface{}
	f   func([]string, io.Writer) error
	h   func(io.Writer)
}

func (this *module_adapter) Run(a []string, w io.Writer) error { return this.f(a, w) }
func (this *module_adapter) Close() error                      { return nil }
func (this *module_adapter) Help(w io.Writer)                  { this.h(w) }

// Module helps with building command-line applications of the form
// <command> <module> <flags...>
type Module interface {
	io.Closer

	Help(io.Writer)
	Run([]string, io.Writer) error
}

func ListModules() []string {
	lock.Lock()
	defer lock.Unlock()

	l := []string{}
	for v, _ := range modules {
		l = append(l, v)
	}
	sort.Strings(l)
	return l
}

func VisitModules(f func(string, Module)) {
	lock.Lock()
	defer lock.Unlock()

	for k, vf := range modules {
		v, _ := vf()
		f(k, v)
	}
}

func GetModule(key string) (Module, bool) {
	lock.Lock()
	defer lock.Unlock()

	cf, has := modules[key]
	if has {
		v, p := cf()
		policies[key] = flag.ErrorHandling(p)
		return v, true
	}
	return nil, false
}

func RunModule(key string, module Module, args []string, w io.Writer) {
	flagSet := flag.NewFlagSet(key, flag.ContinueOnError)
	flagSet.Usage = func() {
		module.Help(os.Stderr)
		flagSet.SetOutput(os.Stderr)
		flagSet.PrintDefaults()
	}
	if adapter, is := module.(*module_adapter); is {
		gflag.RegisterFlags(key, adapter.obj, flagSet)
	} else {
		gflag.RegisterFlags(key, module, flagSet)
	}
	err := flagSet.Parse(args)
	if err != nil {
		handle(err, flag.ExitOnError)
	} else {

		// We make it possible for the module to ask to reparse the flags again.
		// This gives us the ablility to layer flags on top of config data after
		// config template has been applied to an object.
		// We conveniently use function and closures to store copies of the flagSet and args.
		reparseLock.Lock()
		reparseFlags[reflect.TypeOf(module)] = func() {
			// this should be fine here since the first time we parsed ok.
			flagSet.Parse(args)
		}
		reparseLock.Unlock()

		policy, has := policies[key]
		if !has {
			policy = flag.PanicOnError
		}
		handle(module.Run(flagSet.Args(), w), policy)
		handle(module.Close(), policy)
	}
}

// Re-apply the flag settings to the module that was bound earlier.  This is done by matching
// the type of the module and use the bound flagSet and reparse it again.
func ReparseFlags(module Module) {
	reparseLock.Lock()
	defer reparseLock.Unlock()
	if reparse, has := reparseFlags[reflect.TypeOf(module)]; has {
		reparse()
		return
	} else {
		panic(fmt.Errorf("Module not registered for flag parsing."))
	}
}

func handle(err error, handling flag.ErrorHandling) {
	if err != nil {
		switch handling {
		case flag.ContinueOnError:
		case flag.PanicOnError:
			panic(err)
		case flag.ExitOnError:
			if err != flag.ErrHelp {
				fmt.Fprintf(os.Stderr, "Error:", err.Error())
			}
			os.Exit(2)
		}
	}
}
