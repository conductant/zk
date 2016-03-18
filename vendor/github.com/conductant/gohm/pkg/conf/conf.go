// Package for managing configuration objects.
// This works by fetching and overlaying multiple templates onto an object.
package conf

import (
	"bytes"
	"fmt"
	"github.com/conductant/gohm/pkg/encoding"
	gtemplate "github.com/conductant/gohm/pkg/template"
	"golang.org/x/net/context"
	"reflect"
	"strings"
	"sync"
	"text/template"
)

var (
	stringType = reflect.TypeOf("")
)

type Conf struct {
	Urls  []string `flag:"conf.url, Url to fetch for configuration data"`
	model map[string]interface{}

	// Finaly image of the applied and layered template/ content
	serialized []byte

	// Called when a layer of resource has been applied as template
	OnDoneExecuteLayer func(c *Conf, url string, result []byte, err error) `json:"-" yaml:"-"`

	// Called when a layer of resource has been unmarshaled onto the generic model
	OnDoneUnmarshalLayer func(c *Conf, url string, err error) `json:"-" yaml:"-"`

	// Called when all the fetching and layering has been applied to a generic model
	OnDoneFetching func(c *Conf) `json:"-" yaml:"-"`

	// Called when the generic document model has been constructed, layered, and serialized
	OnDoneSerialize func(c *Conf, err error) `json:"-" yaml:"-"`

	// Called when the model has been generated to a composite document and unmarshaled onto the target object
	OnDoneUnmarshal func(c *Conf, target interface{}) `json:"-" yaml:"-"`

	lock sync.Mutex
}

func (this *Conf) Document() []byte {
	buff := bytes.NewBuffer(this.serialized)
	return buff.Bytes()
}

func (this *Conf) Model() map[string]interface{} {
	// deep copy
	copy := map[string]interface{}{}
	for k, v := range this.model {
		copy[k] = v
	}
	return copy
}

type fn struct {
	name     string
	function interface{}
	parent   fmap
}

type fmap map[string]*fn

func NewFuncMap() fmap {
	return fmap{}
}

func (f fmap) Bind(name string) *fn {
	f[name] = &fn{name: name, parent: f}
	return f[name]
}

func (f fmap) Build() template.FuncMap {
	fm := template.FuncMap{}
	for k, v := range f {
		fm[k] = v.function
	}
	return fm
}

func (f *fn) To(fn interface{}) fmap {
	f.function = fn
	return f.parent
}

// Run the configuration specified in conf against the target.  This will run a series
// of template fetching, applying templates, unmarshaling of the final applied template
// onto the given target object.  After unmarshaling is done, the target's fields are
// examined one by one, by tag, and fields that have template as values are then applied.
func Configure(ctx context.Context, conf Conf, target interface{}, optionalFuncs ...template.FuncMap) error {
	conf.lock.Lock()
	defer conf.lock.Unlock()

	initialData := ContextGetInitialData(ctx)
	if initialData != nil {
		gtemplate.ContextPutTemplateData(ctx, initialData)
	}

	contentType := ContextGetConfigDataType(ctx)

	// Generate a list of functions that will escape the template strings
	// Ex.  "secret" : "{{var "zk://host/path/to/secret"}}"
	// This string will be escaped so that the evaluation happens after the unmarshal step by field tag
	funcs := template.FuncMap{}
	for _, fns := range optionalFuncs {
		funcs = gtemplate.MergeFuncMaps(funcs, fns)
	}

	// Note here we generate escaped versions of function to override what's provided.
	// The actual funcs are used in the struct field-by-field step, for those that are marked by tags.
	stubs := gtemplate.MergeFuncMaps(funcs, generateEscapeFuncsFromFieldTag(target))

	conf.model = map[string]interface{}{}
	for _, url := range conf.Urls {

		// Fetch the config data and execute as if it were template
		applied, err := gtemplate.Execute(ctx, url, stubs)

		if conf.OnDoneExecuteLayer != nil {
			conf.OnDoneExecuteLayer(&conf, url, applied, err)
		}

		if err != nil {
			return err
		}

		// Unmarshal to an intermediate representation
		buff := bytes.NewBuffer(applied)
		err = encoding.Unmarshal(contentType, buff, conf.model)

		if conf.OnDoneUnmarshalLayer != nil {
			conf.OnDoneUnmarshalLayer(&conf, url, err)
		}

		if err != nil {
			return err
		}
	}

	if conf.OnDoneFetching != nil {
		conf.OnDoneFetching(&conf)
	}

	// Now marshal the aggregated model to a buffer and then unmarshal it back to the typed struct
	serialized := new(bytes.Buffer)
	err := encoding.Marshal(contentType, serialized, conf.model)
	if err != nil {
		return err
	}
	conf.serialized = serialized.Bytes()
	err = encoding.Unmarshal(contentType, bytes.NewBuffer(conf.serialized), target)

	if conf.OnDoneSerialize != nil {
		conf.OnDoneSerialize(&conf, err)
	}

	if err != nil {
		return err
	}

	if conf.OnDoneUnmarshal != nil {
		conf.OnDoneUnmarshal(&conf, target)
	}

	// Now look for fields with struct tags and apply the actual templates
	return evalStructFieldTemplates(target, funcs)
}

// For all the struct fields that are tagged with `conf`, evaluate the content as a
// template, against the context object of the target itself.  This is where actual
// funcMaps, instead of the escaped version are used.
func evalStructFieldTemplates(target interface{}, funcs template.FuncMap) error {
	t := reflect.TypeOf(target).Elem()
	v := reflect.Indirect(reflect.ValueOf(target))
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fv := v.Field(i)
		// See https://golang.org/ref/spec#Uniqueness_of_identifiers
		exported := f.PkgPath == ""
		if exported && f.Type.ConvertibleTo(stringType) {
			spec := f.Tag.Get("conf")
			if spec == "" {
				continue
			}

			// The field value is a template -- evaluate
			tpl := fv.Convert(stringType).Interface().(string)
			applied, err := gtemplate.Apply([]byte(tpl), target, funcs)
			if err != nil {
				return err
			}
			fv.Set(reflect.ValueOf(string(applied)))
		}
	}
	return nil
}

// For each struct field tag that is `conf:"funcName1,funcName2"` An template escape function
// is generated and added to the funcmap.  This is so we can support template functions that actually
// don't evaluate during the initial template execution and delay until later we have the fields of
// the struct set after unmarshal.
func generateEscapeFuncsFromFieldTag(target interface{}) template.FuncMap {
	fmap := template.FuncMap{}
	t := reflect.TypeOf(target).Elem()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// See https://golang.org/ref/spec#Uniqueness_of_identifiers
		exported := field.PkgPath == ""
		if exported && field.Type.ConvertibleTo(stringType) {

			tag := field.Tag
			spec := tag.Get("conf")
			if spec == "" {
				continue
			}

			p := strings.Split(spec, ",")
			for _, n := range p {
				fmap[n] = escapeFunc(n)
			}
		}
	}
	return fmap
}

var (
	// reservedWords in Go template: https://golang.org/pkg/text/template/
	reservedWords = map[string]int{"template": 1, "range": 1, "block": 1, "with": 1, "if": 1}
)

// This function when added to the funcMap will simply escape all its arguments.  This is so
// that we can preserve some template strings and not have them applied until later on.
func escapeFunc(funcName string) interface{} {
	if _, has := reservedWords[funcName]; has {
		panic(fmt.Errorf("reserved word:%s", funcName))
	}

	return func(args ...interface{}) (string, error) {
		all := ""
		for _, a := range args {
			switch a := a.(type) {
			case string:
				all = all + " \"" + a + "\""
			default:
				all = all + " " + fmt.Sprint(a)
			}
		}
		return "{{" + funcName + all + "}}", nil
	}
}
