package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	if err := receive(); err != nil {
		log.Fatal(err)
	}
}

func receive() error {
	args := []string{
		"-rtsp_transport", "tcp",
		"-i", "rtsp://192.168.1.2:7447/123456abcd",
		"-acodec", "copy",
		"-vcodec", "copy",
		"-f", "mpegts",
		"-",
	}
	cmd := exec.Command("ffmpeg", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}
	defer cmd.Process.Kill()

	br := bufio.NewReader(stdout)

	const packetLen = 188
	pkt := make([]byte, packetLen)

	everySecond := time.NewTicker(1 * time.Second)

	// TODO: replace with loop
	o, err := os.Create("file0.ts")
	if err != nil {
		return err
	}
	defer o.Close()

	bw := bufio.NewWriter(o)

	n := 0
	for {
		if _, err := io.ReadFull(br, pkt); err != nil {
			return err
		}

		if _, err := bw.Write(pkt); err != nil {
			return err
		}

		select {
		case <-everySecond.C:
			log.Print("Flushing")
			bw.Flush()
			o.Close()
			n++
			o, err = os.Create(fmt.Sprintf("file%08d.ts", n))
			if err != nil {
				return err
			}
			bw = bufio.NewWriter(o)
		default:
		}
	}
}
