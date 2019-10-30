package main

import (
	"log"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/client"
	"github.com/open-bastion/open-bastion/internal/ingress"
	//"github.com/open-bastion/open-bastion/internal/auth"
	//"github.com/open-bastion/open-bastion/internal/egress"
	//"golang.org/x/crypto/ssh"
	//"net"
	//"strconv"
)

func main() {
	// hello, world
	log.Println("wake up open bastion")

	// read configuration
	log.Println("search for configuration")
	configPath, err := config.GetConfigurationPath()
	if (err != nil) {
		log.Fatalln(err)
	}

	log.Println("load configuration found at "+configPath)
	var configuration config.Config
	err = configuration.ParseConfig(configPath)
	if (err != nil) {
		log.Fatalln(err)
	}

	// create the client channel
	clientChannel := make(chan client.Client)

	// start ingresses
	var ingressSSH ingress.Ingress
	err = ingressSSH.SetConfigration(configuration)
	if (err != nil) {
		log.Fatalln(err)
	}
	go ingress.StartSSHServer(clientChannel)

//	var sshServer ingress.Ingress
//	var auth auth.Auth
//
//
//	auth.ReadAuthorizedKeysFile(config.AuthorizedKeysFile)
//	sshServer.ConfigSSHServer(auth.AuthorizedKeys, "/home/alex/Documents/open-bastion/bastion.key")
//
//	// Once a ServerConfig has been configured, connections can be
//	// accepted.
//	log.Println("listen for incoming connections")
//	listener, err := net.Listen("tcp", config.ListenAddress+":"+strconv.Itoa(config.ListenPort))
//	if err != nil {
//		log.Fatal("failed to listen for connection: ", err)
//	}
//
//	for {
//		log.Println("accept new connection")
//		client, err := listener.Accept()
//		if err != nil {
//			log.Fatal("failed to accept incoming connection: ", err)
//		}
//
//		go func(c *net.Conn, sshConfig *ssh.ServerConfig) {
//			// Before use, a handshake must be performed on the incoming
//			// net.Conn.
//			hsConn, chans, reqs, err := ssh.NewServerConn(*c, sshConfig)
//			if err != nil {
//				log.Fatal("failed to handshake: ", err)
//			}
//			log.Printf("logged in with key %s", hsConn.Permissions.Extensions["pubkey-fp"])
//
//			// The incoming Request channel must be serviced.
//			go ssh.DiscardRequests(reqs)
//
//			// Service the incoming Channel channel.
//			for newChannel := range chans {
//				log.Printf("channel opened (type=%s)", newChannel.ChannelType())
//				if newChannel.ChannelType() != "session" {
//					newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
//					continue
//				}
//				channel, _, err := newChannel.Accept()
//				if err != nil {
//					log.Fatalf("Could not accept channel: %v", err)
//				}
//
//				// jump to new connection
//				egress.DialSSH(channel, "alex", "127.0.0.1", 22)
//			}
//		}(&client, sshServer.SSHServerConfig)
//	}
}
