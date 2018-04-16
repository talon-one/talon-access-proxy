package microhelpers

import (
	"os"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/pflag"
)

func ParseUint32(flags []string, envs []string, defaultValue uint32, args []string) (uint32, error) {
	var unsetVal uint32
	v := defaultValue

	for _, env := range envs {
		if e := os.Getenv(env); len(e) > 0 {
			v = cast.ToUint32(e)
			break
		}
		if e := os.Getenv(strings.ToUpper(env)); len(e) > 0 {
			v = cast.ToUint32(e)
			break
		}
		if e := os.Getenv(strings.ToLower(env)); len(e) > 0 {
			v = cast.ToUint32(e)
			break
		}
	}
	var flagPtr uint32
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	for _, flag := range flags {
		flag := strings.ToLower(flag)
		if len(flag) > 1 {
			flagSet.Uint32Var(&flagPtr, flag, v, "")
		} else {
			flagSet.Uint32VarP(&flagPtr, "", flag, v, "")
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
