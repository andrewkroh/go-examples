package main

import (
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
)

func init() {
	flag.StringVar(&username, "u", os.Getenv("USER"), "username")
	flag.StringVar(&password, "p", "", "password")
	flag.StringVar(&sshAddress, "addr", "", "ssh address")
}

func main() {
	flag.Parse()

	cmdr, err := ssh.NewClient(sshAddress, username, password)
	if err != nil {
		log.Fatal(err)
	}
	defer cmdr.Close()

	conf, err := cmdr.Command("show running-config")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Running config:")
	fmt.Println(conf)

	cl, err := client.New(cmdr)
	if err != nil {
		log.Fatal(err)
	}
	defer cl.Close()

	accessLists, err := cl.ACLs()
	if err != nil {
		log.Fatal(err)
	}

	json.NewEncoder(os.Stdout).Encode(accessLists)
}
