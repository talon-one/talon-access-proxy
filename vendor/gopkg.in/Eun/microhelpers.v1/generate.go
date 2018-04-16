// +build -ignore

package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

const tmpl = `package microhelpers

import (
	"os"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/pflag"
)

func Parse{{.Type}}(flags []string, envs []string, defaultValue {{.NativeType}}, args []string) ({{.NativeType}}, error) {
	var unsetVal {{.NativeType}}
	v := defaultValue

	for _, env := range envs {
		if e := os.Getenv(env); len(e) > 0 {
			v = cast.To{{.Type}}(e)
			break
		}
		if e := os.Getenv(strings.ToUpper(env)); len(e) > 0 {
			v = cast.To{{.Type}}(e)
			break
		}
		if e := os.Getenv(strings.ToLower(env)); len(e) > 0 {
			v = cast.To{{.Type}}(e)
			break
		}
	}
	var flagPtr {{.NativeType}}
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	for _, flag := range flags {
		flag := strings.ToLower(flag)
		if len(flag) > 1 {
			flagSet.{{.Type}}Var(&flagPtr, flag, v, "")
		} else {
			flagSet.{{.Type}}VarP(&flagPtr, "", flag, v, "")
		}
	}

	if err := flagSet.Parse(args); err != nil {
		return v, nil
	}
	if flagPtr != unsetVal {
		return flagPtr, nil
	}
	return v, nil
}
`

func generate(typ string) {
	f, err := os.Create(fmt.Sprintf("parse_%s.go", strings.ToLower(typ)))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	type context struct {
		Type       string
		NativeType string
	}

	template.Must(template.New("").Parse(tmpl)).Execute(f, &context{
		Type:       typ,
		NativeType: strings.ToLower(typ),
	})
}

func main() {
	argc := len(os.Args)
	if argc <= 1 {
		panic("No arguments passed")
	}

	for i := 1; i < len(os.Args); i++ {
		generate(os.Args[i])
	}
}
