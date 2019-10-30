package main

import (
	"github.com/open-bastion/open-bastion/internal/auth"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/egress"
	"github.com/open-bastion/open-bastion/internal/ingress"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"strconv"
)

func main() {
	var sshServer ingress.Ingress
	var config config.Config
	var auth auth.Auth

	config.ParseConfig("/home/alex/Documents/open-bastion/configs/open-bastion-conf.json")

	auth.ReadAuthorizedKeysFile(config.AuthorizedKeysFile)
	sshServer.ConfigSSHServer(auth.AuthorizedKeys, "/home/alex/Documents/open-bastion/bastion.key")

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	log.Println("listen for incoming connections")
	listener, err := net.Listen("tcp", config.ListenAddress+":"+strconv.Itoa(config.ListenPort))
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}

	for {
		log.Println("accept new connection")
		client, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to accept incoming connection: ", err)
		}

		go func(c *net.Conn, sshConfig *ssh.ServerConfig) {
			// Before use, a handshake must be performed on the incoming
			// net.Conn.
			hsConn, chans, reqs, err := ssh.NewServerConn(*c, sshConfig)
			if err != nil {
				log.Fatal("failed to handshake: ", err)
			}
			log.Printf("logged in with key %s", hsConn.Permissions.Extensions["pubkey-fp"])

			// The incoming Request channel must be serviced.
			go ssh.DiscardRequests(reqs)

			// Service the incoming Channel channel.
			for newChannel := range chans {
				log.Printf("channel opened (type=%s)", newChannel.ChannelType())
				if newChannel.ChannelType() != "session" {
					newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
					continue
				}
				channel, _, err := newChannel.Accept()
				if err != nil {
					log.Fatalf("Could not accept channel: %v", err)
				}

				// jump to new connection
				egress.DialSSH(channel, "alex", "127.0.0.1", 22)
			}
		}(&client, sshServer.SSHServerConfig)
	}
}
