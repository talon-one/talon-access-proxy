// +build -ignore

package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"strings"
)

func main() {
	config, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}

	var configBuffer bytes.Buffer
	if _, err := io.Copy(&configBuffer, config); err != nil {
		panic(err)
	}

	if err := config.Close(); err != nil {
		panic(err)
	}

	var buffer bytes.Buffer

	if _, err := fmt.Fprintf(&buffer, `
package main
	func init() {
		configFile = %s
	}`, fmt.Sprintf("`%s`", strings.TrimSpace(configBuffer.String()))); err != nil {
		panic(err)
	}

	// pretty print
	set := token.NewFileSet()
	astFile, err := parser.ParseFile(set, "", buffer.String(), 0)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse package:", err))
		os.Exit(1)
	}

	f, err := os.Create("generated_consts.go")
	if err != nil {
		panic(err)
	}
	if err := printer.Fprint(f, set, astFile); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}
