package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	username string
	password string
	host     string
)

func init() {
	flag.StringVar(&username, "u", "", "username")
	flag.StringVar(&password, "p", "", "password")
	flag.StringVar(&host, "h", "", "host")
}

func main() {
	flag.Parse()

	conf := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			log.Printf("Key for %v (%v): %v", hostname, remote.String(), hex.EncodeToString(key.Marshal()))
			return nil
		},
	}
	conf.SetDefaults()
	conf.Ciphers = append(conf.Ciphers, "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc")

	// Connect to the remote server and perform the SSH handshake.
	client, err := ssh.Dial("tcp", host+":22", conf)
	if err != nil {
		log.Fatal("Failed to establish SSH connection", err)
	}

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}
	defer session.Close()

	sh, err := newShell(session)
	if err != nil {
		log.Fatal("failed opening session", err)
	}

	if err = sh.Command("show clock detail"); err != nil {
		log.Fatal("failed running show clock", err)
	}

	if err = sh.Command("terminal length 0"); err != nil {
		log.Fatal("failed running show clock", err)
	}

	if err = sh.Command("show startup-config"); err != nil {
		log.Fatal("failed running show clock", err)
	}

	if err = sh.Command("logout"); err != nil {
		log.Fatal("failed while logging out")
	}

	sh.sess.Wait()
	if err = sh.Close(); err != nil {
		log.Fatal("failed closing", err)
	}
	//stdin.Write([]byte("N\n"))
}

type shell struct {
	w    io.Writer
	r    io.Reader
	sess *ssh.Session
	wg   sync.WaitGroup
	line chan string
	errs chan error
}

func newShell(session *ssh.Session) (*shell, error) {
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	session.Stderr = os.Stderr

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdout = io.TeeReader(stdout, os.Stdout)

	stdin, err := session.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err = session.RequestPty("vt100", 0, 200, modes); err != nil {
		return nil, err
	}

	if err = session.Shell(); err != nil {
		return nil, err
	}

	s := &shell{
		sess: session,
		w:    stdin,
		r:    stdout,
		line: make(chan string, 8196),
		errs: make(chan error, 1),
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(s.line)
		defer log.Println("Closing stdout reader.")

		lineReader := bufio.NewScanner(s.r)
		for lineReader.Scan() {
			line := lineReader.Text()
			//log.Printf("Read '%v'", line)
			//line = strings.TrimSpace(line)
			if line != "" {
				s.line <- line
			}
		}
		if err := lineReader.Err(); err != nil {
			log.Println(err)
			s.errs <- err
		}
	}()

	return s, nil
}

func (s *shell) Close() error {
	err := s.sess.Close()
	s.wg.Wait()
	return err
}

func (s *shell) Command(cmd string) error {
	_, err := s.w.Write(append([]byte(cmd), '\n'))
	if err != nil {
		return err
	}

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case line, ok := <-s.line:
			if !ok {
				return nil
			}
			fmt.Println(line)
		case <-timeout.C:
			return nil
		case err = <-s.errs:
			return err
		}
	}
	return nil
}

func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}
