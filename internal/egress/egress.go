package egress

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// BackendConn contains the informations to establish a connection to a backend
type BackendConn struct {
	User string
	Host string
	Port int
}

// DialSSH contact the destination backend server
func DialSSH(channel ssh.Channel, bc BackendConn) error {
	defer channel.Close()

	cb := func() (string, error) {
		return "", nil
	}

	config := &ssh.ClientConfig{
		User: bc.User,
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(cb),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshConn, err := ssh.Dial("tcp", bc.Host+":"+strconv.Itoa(bc.Port), config)
	if err != nil {
		return errors.New("Error dialing backend : " + err.Error())
	}
	defer sshConn.Close()

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := sshConn.NewSession()
	if err != nil {
		return errors.New("Error creating new session : " + err.Error())
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return errors.New("Error getting session stdin : " + err.Error())
	}

	go func() {
		//io.Copy(stdin, channel)
		log := make(chan []byte)
		defer close(log)

		go func(log chan []byte) {
			for {
				txt, ok := <-log

				if !ok {
					break
				}

				fmt.Printf(string(txt))
			}
		}(log)

		copy(stdin, channel, log)
	}()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return errors.New("Error getting session stdout : " + err.Error())
	}

	go func() {
		//io.Copy(channel, stdout)
		copy(channel, stdout, nil)
	}()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return errors.New("Error requestion pseudo terminal : " + err.Error())
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		return errors.New("Error starting shell : " + err.Error())
	}

	log.Println("Shell started, waiting command")
	err = session.Wait()
	if err != nil {
		if err, ok := err.(*ssh.ExitError); ok {
			log.Printf("Command exited with: %v", err)
		} else {
			log.Printf("Failed to start command: %v", err)
		}
	}

	return nil
}

// ParseBackendInfo takes a array containing our payload command and returns
// a BackendConn struct with the required infos to call DialSSH
func ParseBackendInfo(payload string) (bc BackendConn, err error) {
	command := strings.Split(payload, " ")

	if command == nil || len(command) < 1 {
		return bc, errors.New("Invalid payload")
	}

	reg := regexp.MustCompile(`^.+@.+$`)
	regPort := regexp.MustCompile(`-p`)

	for i, s := range command {
		if reg.Match([]byte(s)) {
			arr := strings.Split(s, `@`)

			if net.ParseIP(arr[1]) == nil {
				return bc, errors.New("Could not parse host")
			}

			bc.User = arr[0]
			bc.Host = arr[1]
		} else if regPort.Match([]byte(s)) {
			if i+1 < len(command) {
				port, err := strconv.Atoi(command[i+1])

				if err != nil {
					return bc, errors.New("Invalid port option")
				}

				if port > 65535 || port < 0 {
					return bc, errors.New("Invalid port option")
				}

				bc.Port = port
			} else {
				return bc, errors.New("Invalid port option")
			}
		}
	}

	if bc.Port == 0 {
		bc.Port = 22
	}

	if bc.User == "" || bc.Host == "" {
		return bc, errors.New("Could not parse backend parameters")
	}

	return bc, nil
}

// This function is a reimplementation of the io.Copy function but take a chan where it also write
// the data copied
func copy(dst io.Writer, src io.Reader, log chan []byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	size := 32 * 1024
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf := make([]byte, size)

	for {
		nr, er := src.Read(buf)

		if log != nil {
			log <- buf[0:nr]
		}

		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
