package microhelpers

import (
	"os"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/pflag"
)

func ParseInt(flags []string, envs []string, defaultValue int, args []string) (int, error) {
	var unsetVal int
	v := defaultValue

	for _, env := range envs {
		if e := os.Getenv(env); len(e) > 0 {
			v = cast.ToInt(e)
			break
		}
		if e := os.Getenv(strings.ToUpper(env)); len(e) > 0 {
			v = cast.ToInt(e)
			break
		}
		if e := os.Getenv(strings.ToLower(env)); len(e) > 0 {
			v = cast.ToInt(e)
			break
		}
	}
	var flagPtr int
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	for _, flag := range flags {
		flag := strings.ToLower(flag)
		if len(flag) > 1 {
			flagSet.IntVar(&flagPtr, flag, v, "")
		} else {
			flagSet.IntVarP(&flagPtr, "", flag, v, "")
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
