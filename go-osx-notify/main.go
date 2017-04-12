package main

import "github.com/everdev/mack"

func main() {
	mack.Notify("Motion detected", "Smart Things")
	mack.Say("Motion on the stairs")
}
