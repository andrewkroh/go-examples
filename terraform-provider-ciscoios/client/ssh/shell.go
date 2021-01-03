package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
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

	f, _ := os.Create("dump.txt")

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdout = io.TeeReader(stdout, f)

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
		lineReader.Split(scanLines)
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

func (s *shell) command(cmd string) ([]byte, error) {
	if strings.TrimSpace(cmd) == "" {
		return nil, nil
	}

	_, err := s.w.Write(append([]byte(cmd), '\n'))
	if err != nil {
		return nil, fmt.Errorf("failed to write to ssh shell: %w", err)
	}

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	var sawCmd bool
loop:
	for {
		select {
		case line, ok := <-s.line:
			if !ok {
				break loop
			}
			if strings.HasPrefix(line, "% ") {
				errBuf.WriteString(line)
				continue loop
			}

			//log.Printf("%q", line)

			if sawCmd && strings.HasSuffix(line, "#") {
				//log.Printf("saw response, done, cmd=%q", cmd)
				break loop
			}

			if line == cmd {
				//log.Printf("saw command, cmd=%q", cmd)
				sawCmd = true
				continue loop
			}
			if strings.HasSuffix(line, "#") {
				//log.Printf("saw # suffix, cmd=%q", cmd)
				continue loop
			}

			buf.WriteString(line)
			buf.WriteRune('\n')
		case <-timeout.C:
			log.Println("timeout")
			break loop
		case err = <-s.errs:
			log.Println("error", err)
			return nil, err
		}
	}

	if errBuf.Len() > 0 {
		return nil, fmt.Errorf("error while executing command %q: %q", cmd, errBuf.String())
	}
	return buf.Bytes(), nil
}

func (s *shell) Command(cmd string) ([]byte, error) {
	parts := strings.Split(cmd, "\n")

	buf := new(bytes.Buffer)
	for _, cmdPart := range parts {
		out, err := s.command(cmdPart)
		if err != nil {
			return nil, err
		}
		buf.Write(out)
	}
	return buf.Bytes(), nil
}

func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for # that indicates its back at the prompt:
	// sw01#
	// sw01(config)#
	if i := bytes.IndexAny(data, "#\n"); i >= 0 {
		switch data[i] {
		case '#':
			return i + 1, data[:i+1], nil
		case '\n':
			// We have a full newline-terminated line.
			return i + 1, dropCR(data[0:i]), nil
		}
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
