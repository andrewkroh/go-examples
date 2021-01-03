// Package client provides an interface to Cisco IOS devices. The implementation
// of the connection and communication is handled separately through the
// Commander interface.
package client

//go:generate ragel -Z -G1 accesslist_parse.go.rl -o accesslist_parse.go
//go:generate goimports -l -w accesslist_parse.go

//go:generate ragel -V -p accesslist_parse.go.rl -o accesslist_parse.dot
//go:generate dot -T svg accesslist_parse.dot -o accesslist_parse.svg
//go:generate rm accesslist_parse.dot
