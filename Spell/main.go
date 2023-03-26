package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"

	"plramos.win/9fans/acme"
)

// TODO:
// 	+ add slang env variable to select language to spell
// 	+ add filters like latex and markdown
// 	+ Show commnand?

var DocID = os.Getenv("winid")

type misspell struct {
	word       string
	wincontent string
	q0, q1     int
}

func main() {
	var err error
	var misspells []misspell

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

	offset, _, err := windoc.SelectionAddr()

	courpusraw := windoc.Selection()
	if err != nil {
		log.Fatalf("Could not read content: %v", err)
	}

	cmd := exec.Command("aspell", "-a")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, string(courpusraw))
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	br := bytes.NewReader(out)
	aspellout := bufio.NewScanner(br)

	sr := strings.NewReader(courpusraw)
	courpus := bufio.NewScanner(sr)
	courpus.Scan()

	qconsumed := 0

	aspellout.Scan() // discard first output line
	for aspellout.Scan() {
		// empty line
		if c, _ := utf8.DecodeRune(aspellout.Bytes()); c == '\n' {
			continue
		}
		// new line
		if c, _ := utf8.DecodeRune(aspellout.Bytes()); c == '*' {
			qconsumed += len(courpus.Text()) + 1

			for courpus.Scan() && len(courpus.Text()) == 0 {
				qconsumed++
			}
			continue
		}
		// wrong word with suggestions
		if c, _ := utf8.DecodeRune(aspellout.Bytes()); c == '&' {
			info, corrections, _ := strings.Cut(aspellout.Text(), ":")

			word := strings.Split(info, " ")[1]
			qstr := strings.Split(info, " ")[3]
			q0, err := strconv.Atoi(qstr)
			if err != nil {
				log.Fatalf("invalid address from aspell")
			}

			misspells = append(misspells, misspell{
				word:       word,
				wincontent: "> " + word + "\n" + strings.ReplaceAll(corrections, ", ", "\n"),
				q0:         q0 + offset + qconsumed,
				q1:         q0 + offset + qconsumed + len(word),
			})
		}
		// error but no suggestion
		if c, _ := utf8.DecodeRune(aspellout.Bytes()); c == '#' {
			word := strings.Split(aspellout.Text(), " ")[1]
			qstr := strings.Split(aspellout.Text(), " ")[2]
			q0, err := strconv.Atoi(qstr)
			if err != nil {
				log.Fatalf("invalid address from aspell")
			}
			misspells = append(misspells, misspell{
				word:       word,
				wincontent: "> " + word + "\n~no corrections~",
				q0:         q0 + offset + qconsumed,
				q1:         q0 + offset + qconsumed + len(word),
			})
			continue
		}
	}
	// dummy correction to prevent program exit
	misspells = append(misspells, misspell{
		word:       "",
		wincontent: "~Done~",
		q0:         offset,
		q1:         offset + len(courpusraw),
	})
	wspell, _ := acme.New()
	wspell.Name(docname + "+corrections")
	wspell.Ctl("cleartag")
	wspell.Fprintf("tag", " Next Previous Fix")

NextWord:
	for i := 0; i < len(misspells); i++ {
		if i < 0 {
			i = 0
		}
		wspell.Clear()
		wspell.Fprintf("body", misspells[i].wincontent)
		wspell.Ctl("clean")
		wspell.Addr("0,0")
		wspell.Ctl("dot=addr")
		wspell.Ctl("show")
		if misspells[i].q1 > len(courpusraw) {
			log.Fatal("upsy: out of range")
		}
		windoc.Addr("#%d,#%d", misspells[i].q0, misspells[i].q1)
		windoc.Ctl("dot=addr")
		for e := range wspell.EventChan() {
			switch e.C2 {
			case 'x': // execute in tag
				if string(e.Text) == "Next" {
					continue NextWord
				}
				if string(e.Text) == "Fix" {
					fixedWrd := strings.TrimSpace(wspell.Selection())
					windoc.Fprintf("data", fixedWrd)
					FixPositions(misspells, i, len(fixedWrd)-len(misspells[i].word))
					continue NextWord
				}
				if string(e.Text) == "Previous" {
					i -= 2
					continue NextWord
				}
				if string(e.Text) == "Del" {
					os.Exit(0)
				}
				wspell.WriteEvent(e)
			case 'X':
				fixedWrd := strings.TrimSpace(wspell.Selection())
				windoc.Fprintf("data", fixedWrd)
				FixPositions(misspells, i, len(fixedWrd)-len(misspells[i].word))
				continue NextWord
			default:
				wspell.WriteEvent(e)
			}
		}
	}
}

func FixPositions(misspells []misspell, start int, delta int) {
	if delta == 0 {
		return
	}
	misspells[start].q1 += delta
	start++
	for i := start; i < len(misspells); i++ {
		misspells[i].q0 += delta
		misspells[i].q1 += delta
	}
}
