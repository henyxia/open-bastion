package egress

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"strconv"
)

func DialSSH(channel ssh.Channel, user string, server string, port int) {
	cb := func() (string, error) {
		return "", nil
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(cb),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshConn, err := ssh.Dial("tcp", server+":"+strconv.Itoa(port), config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}
	defer sshConn.Close()

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := sshConn.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		fmt.Errorf("Unable to setup stdin for session: %v", err)
	}

	go func() {
		io.Copy(stdin, channel)
	}()

	stdout, err := session.StdoutPipe()
	if err != nil {
		fmt.Errorf("Unable to setup stdout for session: %v", err)
	}

	go func() {
		io.Copy(channel, stdout)
	}()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	log.Println("request pseudo terminal")
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	log.Println("start shell")
	if err := session.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}

	log.Println("waiting command")
	err = session.Wait()
	if err != nil {
		if err, ok := err.(*ssh.ExitError); ok {
			fmt.Errorf("commad exited with: %v", err)
		} else {
			fmt.Errorf("failed to start command: %v", err)
		}
	}
	log.Println("end")
}
