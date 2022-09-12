package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

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
	if t, err := windoc.ReadAll("tag"); err != nil {
		docname = ""
	} else {
		docname = strings.Fields(string(t))[0]
	}
	q0, q1, err := windoc.SelectionAddr()
	winspell, err := acme.New()
	if err != nil {
		log.Fatalf("Could not open new acme window: %v", err)
	}

	winspell.Name(fmt.Sprintf("%s+Spell", docname))
	winspell.Ctl("cleartag")
	winspell.Fprintf("body", "%d, %d", q0, q1)
	winspell.Ctl("clean")
}