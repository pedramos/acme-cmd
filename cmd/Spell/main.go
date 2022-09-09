package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"9fans.net/go/acme"
)

var DocID = os.Getenv("winid")

func main() {
	id, _ := strconv.Atoi(DocID)
	windoc, _ := acme.Open(id, nil)
	if windoc == nil {
		log.Fatalf("Not running in acme")
	}
	var docname string
	if t, err := windoc.ReadAll("tag"); err != nil {
		docname = ""
	} else {
		docname = strings.Fields(string(t))[0]
	}
	windoc.Ctl("addr=dot")
	d, _ := windoc.ReadAll("xdata")
	fmt.Print(d)
	q0, q1, err := windoc.ReadAddr()
	if err != nil {
		log.Fatalf("Could not set address from selection: %v", err)
	}

	if q0 == q1 {
		windoc.Addr(",")
	}

	data, err := windoc.ReadAll("xdata")
	if err != nil {
		log.Fatalf("Could not get selected text: %v", err)
	}
	winspell, err := acme.New()
	if err != nil {
		log.Fatalf("Could not open new acme window: %v", err)
	}

	winspell.Name(fmt.Sprintf("%s+Spell", docname))
	winspell.Ctl("cleartag")
	winspell.Fprintf("body", "%s\n==\n", data)
	winspell.Fprintf("body", "%d, %d", q0, q1)
	winspell.Ctl("clean")
}