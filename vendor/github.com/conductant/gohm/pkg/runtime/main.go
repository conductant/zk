package runtime

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/conductant/gohm/pkg/command"
	cf "github.com/conductant/gohm/pkg/flag"
	"github.com/conductant/gohm/pkg/version"
	"io"
	"os"
	"strings"
)

// Run the command line main().  Note that the client main go program must import the
// necessary packages (e.g. import _ pkg/a/b/c) where the packages will register the
// module supported by the program.
func Main() {

	buildInfo := version.BuildInfo()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", buildInfo.Notice())
		fmt.Fprintf(os.Stderr, "FLAGS:\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "MODULES:\n\n")
		showHelp(os.Stderr)
	}

	flag.Parse()
	buildInfo.HandleFlag()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return
	}

	key := args[0]
	if len(args) > 1 {
		args = args[1:]
	} else {
		args = []string{}
	}

	module, has := command.GetModule(key)
	if !has {
		fmt.Fprintf(os.Stderr, "%s\n\n", os.Args[0])
		showHelp(os.Stderr)
		return
	}
	command.RunModule(key, module, args, os.Stdout)
}

func showHelp(out io.Writer) {
	command.VisitModules(func(v string, module command.Module) {
		fmt.Fprintf(out, "%s\n", v)

		buff := new(bytes.Buffer)
		module.Help(buff)

		for _, line := range strings.Split(buff.String(), "\n") {
			fmt.Fprintf(out, "  %s\n", line)
		}
		// show flags
		fs := flag.NewFlagSet(v, flag.PanicOnError)
		cf.RegisterFlags(v, module, fs)
		fs.PrintDefaults()
	})
}
