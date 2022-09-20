package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"pedrolorgaramos.win/s/9fans-go/acme"
)

var DocID = os.Getenv("winid")

func main() {
	id, _ := strconv.Atoi(DocID)
	windoc, _ := acme.Open(id, nil)
	if windoc == nil {
		log.Fatalf("Not running in acme")
	}
	var docname string
	if t, err := windoc.ReadAll("tag"); err != nil || t[0] == ' ' {
		docname = ""
	} else {
		docname = strings.Fields(string(t))[0]
	}
	q0, _, err := windoc.SelectionAddr()
	winspell, err := acme.New()
	if err != nil {
		log.Fatalf("Could not open new acme window: %v", err)
	}
	winspell.Name(fmt.Sprintf("%s+Spell", docname))
	winspell.Ctl("cleartag")
	winspell.Write("tag", []byte(" Next Previous Fix"))
	winspell.Ctl("clean")

	xdata, err := windoc.ReadAll("xdata")
	if err != nil {
		log.Fatalf("Could not read content: %v", err)
	}

	r := bytes.NewReader(xdata)
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanRunes)

	var (
		wq0  int = q0
		wq1  int = q0
		word strings.Builder
	)

	for scanner.Scan() {
		r, _ := utf8.DecodeRune(scanner.Bytes())
		if unicode.IsSpace(r) {
			// This only happens if text starts with space
			if word.Len() == 0 {
				wq1++
				wq0 = wq1
				continue
			}
			windoc.Addr("#%d,#%d", wq0, wq1)
			windoc.Ctl("dot=addr")
			spellcheck(winspell, word.String())
			word.Reset()
		}
		if unicode.IsPrint(r) {
			word.WriteRune(r)
			wq1++
		}
		if scanner.Err() == bufio.ErrFinalToken {
			windoc.Addr("#%d,#%d", wq0, wq1)
			windoc.Ctl("dot=addr")
			spellcheck(winspell, word.String())
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read word")
	}
	winspell.Del(true)
}

func spellcheck(w *acme.Win, word string) {
	w.Addr(",")
	w.Write("data", nil)
	w.Fprintf("body", "Checking %s", word)
	w.Ctl("clean")

	for e := range w.EventChan() {
		switch e.C2 {
		case 'x': // execute in tag
			if string(e.Text) == "Next" {
				return
			}
			if string(e.Text) == "Del" {
				w.Del(true)
				os.Exit(0)
			}
		}
	}
}
