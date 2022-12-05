package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/trustmaster/go-aspell"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"pedrolorgaramos.win/s/9fans-go/acme"
)

// TODO: add slang env variable to select language to spell

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
	windoc, _ = acme.Open(id, nil)
	if windoc == nil {
		log.Fatalf("Not running in acme")
	}

	var (
		// wq0 and wq1 marks the start and end of the word to be spellchecked
		wq0 int = q0
		wq1 int = q0

		// word is the buffer which will contain the full word to be spellchecked
		word strings.Builder

		// scanner goes through xdata. It reads rune by rune
		scanner *bufio.Scanner = bufio.NewScanner(r)
	)

	scanner.Split(bufio.ScanRunes)
	for scanner.Scan() {
		r, _ := utf8.DecodeRune(scanner.Bytes())
		if r != '-' && (unicode.IsSpace(r) || unicode.IsPunct(r)) {
			// This only happens if text starts with space
			if word.Len() == 0 {
				wq1++
				wq0 = wq1
				continue
			}
			if err := windoc.Addr("#%d,#%d", wq0, wq1); err != nil {
				log.Fatalf("Error setting addr: %v", err)
			}
			windoc.Ctl("dot=addr")
			if fixedWrd, deltaQ := spellcheck(winspell, word.String()); !(fixedWrd == "" && deltaQ == 0) {
				windoc.Write("data", []byte(fixedWrd))
				wq1 += deltaQ
			}
			word.Reset()

			wq1++
			wq0 = wq1
		}
		if unicode.IsLetter(r) || r == '-' {
			word.WriteRune(r)
			wq1++
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read word")
	}
	if fixedWrd, deltaQ := spellcheck(winspell, word.String()); !(fixedWrd == "" && deltaQ == 0) {
		if err := windoc.Addr("#%d,#%d", wq0, wq1); err != nil {
			log.Fatalf("Error setting addr: %v", err)
		}
		windoc.Write("data", []byte(fixedWrd))
	}
	winspell.Del(true)
}

func spellcheck(w *acme.Win, word string) (fixedWrd string, deltaQ int) {
	w.Addr(",")
	w.Write("data", nil)
	w.Ctl("clean")

	speller, err := aspell.NewSpeller(map[string]string{
		"lang": "en_US",
	})
	if err != nil {
		log.Fatalf("could not open speller: %v", err)
	}
	defer speller.Delete()

	w.Fprintf("body", "> %s\n", word)
	w.Ctl("clean")

	var sugglst []string
	if speller.Check(word) {
		return
	} else if len(speller.Suggest(word)) < 5 {
		sugglst = speller.Suggest(word)
	} else {
		sugglst = speller.Suggest(word)[:5]
	}

	w.Fprintf("body", "%s\n", strings.Join(sugglst, "\n"))
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
			if string(e.Text) == "Fix" {
				fixedWrd = strings.TrimSpace(w.Selection())
				deltaQ = len(fixedWrd) - len(word)
				return
			}
		case 'X':
			fixedWrd = strings.TrimSpace(string(e.Text))
			deltaQ = len(fixedWrd) - len(word)
			return
		}
	}
	return
}
