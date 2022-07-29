package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strings"

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
			w.Fprintf("body", "Error on %s: %v\n", path, err)
			w.Ctl("clean")
		}
		if d.IsDir() {
			dir = d.Name()
			return nil
		}
		w.Fprintf("body", "%s/%s\n", dir, d.Name())
		return nil
	})
	w.Ctl("clean")

	for e := range w.EventChan() {
		switch e.C2 {
		case 'x', 'X': // execute
			// Get Dial Info Add Rm
			if string(e.Text) == "Get" {
				continue
			}
			if string(e.Text) == "Dial" {
				sshDescription := strings.TrimSpace(w.Selection())
				w.Del(true)
				f, err := fileSystem.Open(sshDescription)
				if err != nil {
					w.Fprintf("body", "Cannot open %s: %v\n", sshDescription, err)
					w.Ctl("clean")
				}
				sshWin(f)
			}
		}
	}
}

func sshWin(r io.Reader) {
	scanner := bufio.NewScanner(r)
	var b strings.Builder
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "--end--" {
			break
		}
		b.WriteString(scanner.Text())
		b.WriteRune('\n')
	}

	var (
		host, user, key string
		password        bool
	)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		switch f[0] {
		case "host":
			host = f[1]
		case "user":
			user = f[1]
		case "password":
			password = true
		case "key":
			key = f[1]
		}
	}
	if host == "" || (!password && key == "") || user == "" {
		log.Fatalf("Bad ssh descriptor:\nhost: %s\npassword:%v\nuser:%s", host, password, user)
	}
	var sshCmd []string
	sshCmd = append(sshCmd, "ssh", host, "-l", user)
	if !password {
		sshCmd = append(sshCmd, "-i", key)
	}
	c:= exec.Command("win", sshCmd...)
	c.Start()
}
