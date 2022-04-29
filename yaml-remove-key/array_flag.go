package main

import "strings"

type arrayFlag []string

func (f *arrayFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *arrayFlag) Set(value string) error {
	keys := strings.Split(value, ",")
	for _, k := range keys {
		*f = append(*f, strings.TrimSpace(k))
	}
	return nil
}
