package main

import (
	"bufio"
	"context"
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
var defaultDir = fmt.Sprintf("%s/lib/coms/ssh", HOME)
var sshDir = flag.String("d", defaultDir, "Directory contianing all the ssh connection description")

func main() {
	w, _ := acme.New()
	w.Name("/ssh/+list")
	w.Fprintf("tag", "Get Dial Info Add")
	fileSystem := os.DirFS(*sshDir)
	writeSshEntries(w, fileSystem)

	for e := range w.EventChan() {
		switch e.C2 {
		case 'x': // execute in tag
			// Get Dial Info Add Rm
			if string(e.Text) == "Get" {
				continue
			}
			if string(e.Text) == "Dial" {
				dial(w, e, fileSystem)
			}
			if string(e.Text) == "Del" {
				w.Del(true)
				os.Exit(0)
			}
			if string(e.Text) == "Info" {
				sshConfig := strings.TrimSpace(w.Selection())
				f, err := fileSystem.Open(sshConfig)
				if err != nil {
					w.Fprintf("body", "Cannot open %s: %v\n", sshConfig, err)
					w.Ctl("clean")
				}
				displayInfo(f, sshConfig)
			}
			if string(e.Text) == "Add" {
				createEntry()
			}
		case 'X': // executes in body
			dial(w, e, fileSystem)
		}
	}
}

func createEntry() {
	w, _ := acme.New()
	winName := fmt.Sprintf("%s/", *sshDir)
	w.Name(winName)
	w.Write("body", []byte(`
--end--
host
username
key
password
`))
	w.Ctl("clean")
}

func writeSshEntries(w *acme.Win, fileSystem fs.FS) {

	fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			w.Fprintf("body", "Error on %s: %v\n", path, err)
			w.Ctl("clean")
		}
		if d.IsDir() {
			return nil
		}
		if path != "." {
			w.Fprintf("body", "%s\n", path)
		} else {
			w.Fprintf("body", "%s\n", d.Name())
		}
		return nil
	})
	w.Ctl("clean")
}

func dial(w *acme.Win, e *acme.Event, fileSystem fs.FS) {
	sshConfig := strings.TrimSpace(string(e.Text))
	w.Del(true)
	f, err := fileSystem.Open(sshConfig)
	if err != nil {
		w.Fprintf("body", "Cannot open %s: %v\n", sshConfig, err)
		w.Ctl("clean")
	}
	scanner := bufio.NewScanner(f)
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
	var sshCmd []string = []string{"ssh", host, "-l", user}
	if !password {
		sshCmd = append(sshCmd, "-i", key)
	}
	c := exec.Command("win", sshCmd...)
	c.Start()

	mntPoint := fmt.Sprintf("%s/n/%s", HOME, e.Text)
	os.MkdirAll(mntPoint, 0770)
	fsCmdArgs := []string{
		"-C",
		"-o",
		fmt.Sprintf("IdentityFile=%s", key),
		fmt.Sprintf("%s@%s:", user, host),
		mntPoint,
	}
	ctx, cancel := context.WithCancel(context.Background())
	sshFS := exec.CommandContext(ctx, "sshfs", fsCmdArgs...)
	stderr, _ := sshFS.StderrPipe()
	go io.Copy(os.Stderr, stderr)
	if err := sshFS.Start(); err != nil {
		log.Printf("Could not mount sshfs: %v", err)
	}
	sshDirWin, _ := acme.New()
	sshDirWin.Name(mntPoint)
	sshDirWin.Ctl("get")
	c.Wait()
	cancel()
}

func displayInfo(f fs.File, path string) error {
	w, _ := acme.New()
	w.Name(fmt.Sprintf("/ssh/%s/+info", path))

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "--end--" {
			break
		}
		w.Write("body", scanner.Bytes())
		w.Write("body", []byte("\n"))
	}
	w.Ctl("clean")
	return nil
}
