package main

import (
	"io/fs"
	"log"
	"os"

	"9fans.net/go/acme"
)

func main() {
	w, _ := acme.New()
	w.Name("/remote/+list")

	fileSystem := os.DirFS("/home/pedramos/nokia/cons/9labs/")
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