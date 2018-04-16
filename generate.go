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
	"os/exec"
	"strings"
	"time"
)

func writeVar(w io.Writer, key, value string) {
	if _, err := fmt.Fprintf(w, "%s = \"%s\"\n", key, value); err != nil {
		panic(err)
	}
}

func getVersionHash() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	if err := cmd.Run(); err != nil {
		return "Unknown/CustomBuild"
	}
	return strings.TrimSpace(buffer.String())
}

func getVersion() string {
	cmd := exec.Command("git", "describe", "--abbrev=0", "--tags")
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	if err := cmd.Run(); err != nil {
		return "Unknown/CustomBuild"
	}
	return strings.TrimSpace(buffer.String())
}

type info struct {
	Version     string
	BuildDate   string
	VersionHash string
}

func main() {
	var buffer bytes.Buffer
	if _, err := io.WriteString(&buffer, "package talon_access_proxy\n"); err != nil {
		panic(err)
	}

	info := info{
		Version:     getVersion(),
		VersionHash: getVersionHash(),
		BuildDate:   time.Now().UTC().Format(time.RFC1123),
	}

	io.WriteString(&buffer, "func init() {\n")
	writeVar(&buffer, "VersionHash", info.VersionHash)
	writeVar(&buffer, "Version", info.Version)
	writeVar(&buffer, "BuildDate", info.BuildDate)
	io.WriteString(&buffer, "}")

	// pretty print
	set := token.NewFileSet()
	astFile, err := parser.ParseFile(set, "", buffer.String(), parser.ParseComments)
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
