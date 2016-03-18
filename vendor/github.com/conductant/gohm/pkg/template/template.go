package template

import (
	"bytes"
	"fmt"
	"github.com/conductant/gohm/pkg/resource"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
	"hash/fnv"
	"strconv"
	"text/template"
)

type templateDataContextKey int

var (
	TemplateDataContextKey templateDataContextKey = 1
)

func ContextPutTemplateData(ctx context.Context, data interface{}) context.Context {
	return context.WithValue(ctx, TemplateDataContextKey, data)
}
func ContextGetTemplateData(ctx context.Context) interface{} {
	return ctx.Value(TemplateDataContextKey)
}

func GetKeyForTemplate(tmpl []byte) string {
	hash := fnv.New64a()
	hash.Write(tmpl)
	return strconv.FormatUint(hash.Sum64(), 16)
}

// Generic Apply template.  This is simple convenince wrapper that generates a hash key
// for the template name based on the template content itself.
func Apply(tmpl []byte, data interface{}, funcs ...template.FuncMap) ([]byte, error) {
	fm := template.FuncMap{}
	for _, opt := range funcs {
		fm = MergeFuncMaps(fm, opt)
	}
	t := template.New(GetKeyForTemplate(tmpl)).Funcs(fm)
	t, err := t.Parse(string(tmpl))
	if err != nil {
		return nil, err
	}
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	return buff.Bytes(), err
}

/*
Two-pass apply.  This supports the notion of a define template block where the content is
in YAML. The content in YAML is then parsed and in the second pass any template expressions that
calls the function {{my <var>}} will access the values in the define blocks.

For example:

{{define "app"}}
version: 1.2
image: repo
build: 1234
{{end}}

{{define "host"}}
label: appserver
name: myhost
{{end}}

{
   "image" : "repo/myapp:{{my app.version}}-{{my app.build}}",
   "host" : "{{my host.name}}"
}

Will turn into:

{
   "image" : "repo/myapp:1.2-1234",
   "host" : "myhost"
}
*/
func Apply2(tmpl []byte, data interface{}, funcs ...template.FuncMap) ([]byte, error) {
	fm := template.FuncMap{}
	for _, opt := range funcs {
		fm = MergeFuncMaps(fm, opt)
	}

	// support for my variables via the define block:
	// first pass just escape all the expressions
	fm["my"] = func(v string) string {
		return "{{my '" + v + "'}}"
	}

	t := template.New(GetKeyForTemplate(tmpl)).Funcs(fm)
	t, err := t.Parse(string(tmpl))
	if err != nil {
		return nil, err
	}

	vars := make(map[string]interface{})
	for _, ct := range t.Templates() {
		if t.Name() == ct.Name() {
			// Do not process the main template.  Save that for the next phase
			// where `my` variables are supported.
			continue
		}

		body := new(bytes.Buffer)
		err := ct.Execute(body, data)
		if err != nil {
			return nil, err
		}
		// attempt to parse as YAML.  It's possible that
		// the template may be something else.
		m := make(map[string]interface{})
		if err := yaml.Unmarshal(body.Bytes(), m); err == nil {
			for k, v := range m {
				vars[ct.Name()+"."+k] = v
			}
		} else {
			return nil, err
		}
	}

	// Second pass - now we do not escape the my function.
	fm["my"] = func(k string) string {
		if v, has := vars[k]; has {
			switch v := v.(type) {
			case string:
				return v
			default:
				return fmt.Sprintf("%v", v)
			}
		} else {
			return "?"
		}
	}
	buff := new(bytes.Buffer)
	err = t.Funcs(fm).Execute(buff, data)
	return buff.Bytes(), err
}

// Execute a template at the given uri/url.  The data to be applied to the template should
// be placed in the context via the ContextPutTemplateData() function.
func Execute(ctx context.Context, uri string, funcs ...template.FuncMap) ([]byte, error) {
	data := ContextGetTemplateData(ctx)
	fm := DefaultFuncMap(ctx)
	for _, opt := range funcs {
		fm = MergeFuncMaps(fm, opt)
	}

	// THe url itself can be a template that uses the state of the context object as vars.
	url := uri
	if applied, err := Apply([]byte(uri), data, fm); err != nil {
		return nil, err
	} else {
		url = string(applied)
	}

	body, err := resource.Fetch(ctx, url)
	if err != nil {
		return nil, err
	}
	return Apply2(body, data, fm)
}
