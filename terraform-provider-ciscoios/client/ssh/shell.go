package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

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
		line: make(chan string, 128),
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
			if line != "" {
				s.line <- line
			}
		}
		if err := lineReader.Err(); err != nil {
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

func (s *shell) Command(cmd string) ([]byte, error) {
	_, err := s.w.Write(append([]byte(cmd), '\n'))
	if err != nil {
		return nil, fmt.Errorf("failed to write to ssh shell: %w", err)
	}

	buf := new(bytes.Buffer)
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
loop:
	for {
		select {
		case line, ok := <-s.line:
			if !ok {
				break loop
			}
			buf.WriteString(line)
			buf.WriteRune('\n')
		case <-timeout.C:
			break loop
		case err = <-s.errs:
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
