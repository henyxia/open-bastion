package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
)

// Get default location of a private key
func privateKeyPath() string {
	return os.Getenv("HOME") + "/.ssh/id_rsa"
}

// Get private key for ssh authentication
func parsePrivateKey(keyPath string) (ssh.Signer, error) {
	buff, _ := ioutil.ReadFile(keyPath)
	return ssh.ParsePrivateKey(buff)
}

// Get ssh client config for our connection
// SSH config will use 2 authentication strategies: by key and by password
func makeSshConfig(user, password string) (*ssh.ClientConfig, error) {
	key, err := parsePrivateKey(privateKeyPath())
	if err != nil {
		return nil, err
	}

	config := ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
			ssh.Password(password),
		},
	}

	return &config, nil
}

// Handle local client connections and tunnel data to the remote serverq
// Will use io.Copy - http://golang.org/pkg/io/#Copy
func handleClient(client net.Conn, remote net.Conn) {
	defer client.Close()
	chDone := make(chan bool)

	// Start remote -> local data transfer
	go func() {
		_, err := io.Copy(client, remote)
		if err != nil {
			log.Println("error while copy remote->local:", err)
		}
		chDone <- true
	}()

	// Start local -> remote data transfer
	go func() {
		_, err := io.Copy(remote, client)
		if err != nil {
			log.Println(err)
		}
		chDone <- true
	}()

	<-chDone
}

func main() {
	// Build SSH client configuration
	authorizedKeysMap := map[string]bool{}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCvDoxgUqn0SB0+X6uDY9wbOTnFqmkmd2AITpA8GWHHT4uRVnKiMbcifEy1HrpRJkX4ziDFMXpLQgBIQr9eKnCcCL5qmfOCL7G3X21MvX2LECf8SGF59QarNvjVznLM7bhKuZ8ixmo1BShLrIyFqI2HBGNVVvP800kS3l1CezEMBsLzDv+G/aLN52ZfAh3WZmrqdE8pKcByVWbFA5nnHcsulqESBiAO7ThESTP/H6/5pPnXz+eESXzkEwMYVBE6wJgZ64O1uXIdmb19YfuQUMy1bvzKZuDm5rLIgtlgaTrCmh/5UgIVEXzCeLUSgzcNAFPS5fhgqzZjDrjGvjl2nCwJ alex@localhost.localdomain"))
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

	privateBytes, err := ioutil.ReadFile("/home/alex/Documents/open-bastion/user.key")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config.AddHostKey(private)

	//Listen for incoming SSH conn

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2022")
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}
	client, err := listener.Accept()
	if err != nil {
		log.Fatal("failed to accept incoming connection: ", err)
	}

	// Before use, a handshake must be performed on the incoming
	// net.Conn.
	hsConn, chans, reqs, err := ssh.NewServerConn(client, config)
	if err != nil {
		log.Fatal("failed to handshake: ", err)
	}
	log.Printf("logged in with key %s", hsConn.Permissions.Extensions["pubkey-fp"])

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

		//Need to copy channel io to the new con

		dialSSH(channel)

	}
}

func dialSSH(channel ssh.Channel) {
	server := "127.0.0.1"
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
	sshConn, err := ssh.Dial("tcp", server, config)
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

	// Try to connect channel output to session input
	go func() {
		b := make([]byte, 2048)
		p, err := session.StdinPipe()
		for {
			channel.Read(b)
		}
	}()

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
}