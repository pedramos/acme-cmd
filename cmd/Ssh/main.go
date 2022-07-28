package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"9fans.net/go/acme"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: Ssh -d directory \n")
	os.Exit(2)
}

var HOME = os.Getenv("HOME")
var defaultDir = fmt.Sprintf("%s/lib/coms/", HOME)
var sshDir = flag.String("d", defaultDir, "Directory contianing all the ssh connection description")

func main() {
	w, _ := acme.New()
	w.Name("/ssh/+list")
	w.Fprintf("tag", "Get Dial Info Add Rm")

	fileSystem := os.DirFS(*sshDir)
	var dir string
	w.Ctl("clean")
	fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			w.Write("body", []byte(err.Error()))
			log.Fatal(err)
		}
		if d.IsDir() {
			dir = d.Name()
			return nil
		}
		w.Fprintf("body", "%s/%s\n", dir, d.Name())
		return nil
	})
	w.Ctl("clean")
}