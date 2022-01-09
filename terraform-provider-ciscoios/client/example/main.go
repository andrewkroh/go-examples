package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client"
	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client/ssh"
)

var (
	username   string
	password   string
	sshAddress string
	readOnly   bool
)

func init() {
	flag.StringVar(&username, "u", os.Getenv("USER"), "username")
	flag.StringVar(&password, "p", "", "password")
	flag.StringVar(&sshAddress, "addr", "", "ssh address")
	flag.BoolVar(&readOnly, "ro", true, "treat device as read-only")
}

var testAccessList = client.AccessList{
	ID: "140",
	Rules: []client.AccessListEntry{
		{
			Remark: "Allow established TCP connections.",
		},
		{
			Permit:      true,
			Protocol:    "tcp",
			Source:      "any",
			Destination: "any",
			Established: true,
		},
		{
			Remark: "Allow outgoing TCP/443 (HTTPS) connections.",
		},
		{
			Permit:          true,
			Protocol:        "tcp",
			Source:          "any",
			Destination:     "any",
			DestinationPort: "eq 443",
		},
		{
			Remark: "Deny all other traffic.",
		},
		{
			Permit:      false,
			Protocol:    "ip",
			Source:      "any",
			Destination: "any",
			Log:         true,
		},
	},
}

func main() {
	log.SetOutput(os.Stderr)
	flag.Parse()

	cmdr, err := ssh.NewClient(sshAddress, username, password)
	if err != nil {
		log.Fatal(err)
	}
	defer cmdr.Close()

	cl, err := client.New(cmdr)
	if err != nil {
		log.Fatal(err)
	}
	defer cl.Close()

	accessLists, err := cl.ACLs()
	if err != nil {
		log.Fatal(err)
	}

	data, err := toPrettyJSON(accessLists)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("access-lists:")
	fmt.Println(string(data))

	id, err := client.FreeExtendedAccessListID(accessLists)
	if err != nil {
		log.Fatal(err)
	}

	if readOnly {
		return
	}

	log.Println("Creating demo access-list", id)
	testAccessList.ID = id

	if err = cl.CreateACL(testAccessList); err != nil {
		log.Fatal("Failed to create access-list.", err)
	}
}

func toPrettyJSON(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
