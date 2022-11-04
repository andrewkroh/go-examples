package main

import "github.com/andybrewer/mack"

func main() {
	// On macos popup a notification.
	mack.Notify("Motion detected", "Smart Things")

	// Use text-to-speech on to make announcement on macos.
	mack.Say("Motion on the stairs")
}
