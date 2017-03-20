package main

import (
	"fmt"

	"github.com/mozilla/libaudit-go"
)

func main() {
	s, err := libaudit.NewNetlinkConnection()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	defer s.Close()

	rules, err := libaudit.ListAllRules(s)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	for i, r := range rules {
		fmt.Printf("%04d - %s\n", i, r)
	}
}
