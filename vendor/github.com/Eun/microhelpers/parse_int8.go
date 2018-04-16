package microhelpers

import (
	"os"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/pflag"
)

func ParseInt8(flags []string, envs []string, defaultValue int8, args []string) (int8, error) {
	var unsetVal int8
	v := defaultValue

	for _, env := range envs {
		if e := os.Getenv(env); len(e) > 0 {
			v = cast.ToInt8(e)
			break
		}
		if e := os.Getenv(strings.ToUpper(env)); len(e) > 0 {
			v = cast.ToInt8(e)
			break
		}
		if e := os.Getenv(strings.ToLower(env)); len(e) > 0 {
			v = cast.ToInt8(e)
			break
		}
	}
	var flagPtr int8
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	for _, flag := range flags {
		flag := strings.ToLower(flag)
		if len(flag) > 1 {
			flagSet.Int8Var(&flagPtr, flag, v, "")
		} else {
			flagSet.Int8VarP(&flagPtr, "", flag, v, "")
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
