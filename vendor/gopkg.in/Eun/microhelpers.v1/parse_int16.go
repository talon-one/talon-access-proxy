package microhelpers

import (
	"os"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/pflag"
)

func ParseInt16(flags []string, envs []string, defaultValue int16, args []string) (int16, error) {
	var unsetVal int16
	v := defaultValue

	for _, env := range envs {
		if e := os.Getenv(env); len(e) > 0 {
			v = cast.ToInt16(e)
			break
		}
		if e := os.Getenv(strings.ToUpper(env)); len(e) > 0 {
			v = cast.ToInt16(e)
			break
		}
		if e := os.Getenv(strings.ToLower(env)); len(e) > 0 {
			v = cast.ToInt16(e)
			break
		}
	}
	var flagPtr int16
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	for _, flag := range flags {
		flag := strings.ToLower(flag)
		if len(flag) > 1 {
			flagSet.Int16Var(&flagPtr, flag, v, "")
		} else {
			flagSet.Int16VarP(&flagPtr, "", flag, v, "")
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
