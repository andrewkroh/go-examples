package fydler

import "strings"

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ", ")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
