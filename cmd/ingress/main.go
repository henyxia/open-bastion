package main

import (
	"bufio"
	"fmt"
	//"github.com/open-bastion/open-bastion/internal/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	//args := os.Args

	// serverConfig := new(config.Config)
	// serverConfig.ParseConfig(args[1])

	// Public key authentication is done by comparing
	// the public key of a received connection
	// with the entries in the authorized_keys file.
	// authorizedKeysBytes, err := ioutil.ReadFile("~/.ssh/authorized_keys")
	// if err != nil {
	// 	log.Fatalf("Failed to load authorized_keys, err: %v", err)
	// }

	// authorizedKeysMap := map[string]bool{}
	// for len(authorizedKeysBytes) > 0 {
	// 	pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	authorizedKeysMap[string(pubKey.Marshal())] = true
	// 	authorizedKeysBytes = rest
	// }
	authorizedKeysMap := map[string]bool{}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDNznyqlPdH6Fzyd/Rb0oFSVMQMHK13OeSqCmjWxKF0hVOljGxAL9qm9Sscq/slBknvRCloy6Xmj3tRJtMSp3Oezu+ChRt6qX3W5I/AJbx8iyDmt4zITWKH4DKjiXd7mO5dKG5IeafWysghN1RozJIt+kVQy7ehq3o7R2bDFM/WzSsTiKyM/PYHI9arlWwqqOArI6pz8eI3J2EWYnQMkWAgBb8bhZTExE63G6bKiBYkmFmf1jUlhPhn/cY7s8qm04cAp+vc8GrIMlkUAP0ElmCyWwFW0v9CX9wrdUVBVBx9fJI5+k7KLyiPJD3Zyzi/KCIsJ/uE83waVVgHmwpYayibuGDKh4VZHQGPjNrXz9gSOVxaVVkSylkf2yRaN4zYUHAjiFF9oc1Xk+sdIwtgDQKpqJqwC4GAv8K8TrFl81yxx9rjF6dEfKLCXgqH3EvAPQiM7pw/JgFde+cKZhg7UFUbaiiSJbEGaFP4Sc32UoFdaYBJSBnaF19/b3cODhDMnvE="))
	if err != nil {
		log.Fatal(err)
	}

	authorizedKeysMap[string(pubKey.Marshal())] = true

	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2022")
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}
	nConn, err := listener.Accept()
	if err != nil {
		log.Fatal("failed to accept incoming connection: ", err)
	}

	// Before use, a handshake must be performed on the incoming
	// net.Conn.
	conn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		log.Fatal("failed to handshake: ", err)
	}
	log.Printf("logged in with key %s", conn.Permissions.Extensions["pubkey-fp"])

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of a shell, the type is
		// "session" and ServerShell may be used to present a simple
		// terminal interface.
		log.Printf("channel opened (type=%s)", newChannel.ChannelType())
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, _, err := newChannel.Accept()
		if err != nil {
			log.Fatalf("Could not accept channel: %v", err)
		}

		term := terminal.NewTerminal(channel, "open-bastion> ")

		go func() {
			defer channel.Close()

			server := "localhost"
			port := "22"

			server = server + ":" + port
			user := "foo"
			p := "bar"

			var pass = string(p)
			config := &ssh.ClientConfig{
				User: user,
				Auth: []ssh.AuthMethod{
					// ClientAuthPassword wraps a ClientPassword implementation
					// in a type that implements ClientAuth.
					ssh.Password(pass),
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}
			conn, err := ssh.Dial("tcp", server, config)
			if err != nil {
				panic("Failed to dial: " + err.Error())
			}
			defer conn.Close()

			// Each ClientConn can support multiple interactive sessions,
			// represented by a Session.
			session, err := conn.NewSession()
			if err != nil {
				panic("Failed to create session: " + err.Error())
			}
			defer session.Close()

			// Set IO
			//session.Stdout = os.Stdout
			//session.Stderr = os.Stderr
			out, _ := session.StdoutPipe()
			in, _ := session.StdinPipe()

			// Set up terminal modes
			modes := ssh.TerminalModes{
				ssh.ECHO:          0,     // disable echoing
				ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
				ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
			}

			// Request pseudo terminal
			if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
				log.Fatalf("request for pseudo terminal failed: %s", err)
			}

			// Start remote shell
			if err := session.Shell(); err != nil {
				log.Fatalf("failed to start shell: %s", err)
			}

			// Accepting commands
			// for {
			// 	reader := bufio.NewReader(os.Stdin)
			// 	str, _ := reader.ReadString('\n')
			// 	fmt.Fprint(in, str)
			// }

			go func() {
				for {
					reader := bufio.NewReader(out)
					str, _ := reader.ReadString('\n')
					term.Write([]byte(str))
					fmt.Printf(str)
					time.Sleep(time.Second * 2)
					fmt.Printf("ping")
				}
			}()

			for {
				line, err := term.ReadLine()
				if err != nil {
					break
				}
				if line == "exit" {
					break
				}

				fmt.Fprintf(in, line)

				// term.Write([]byte("got:"))
				// term.Write([]byte(line))
				// term.Write([]byte("\n"))
			}
		}()
	}
}

func test() {

	server := "localhost"
	port := "22"

	server = server + ":" + port
	user := "foo"
	p := "bar"

	var pass = string(p)
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			// ClientAuthPassword wraps a ClientPassword implementation
			// in a type that implements ClientAuth.
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", server, config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}
	defer conn.Close()

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := conn.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	defer session.Close()

	// Set IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	in, _ := session.StdinPipe()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}

	// Accepting commands
	for {
		reader := bufio.NewReader(os.Stdin)
		str, _ := reader.ReadString('\n')
		fmt.Fprint(in, str)
	}

}
