package main

import (
	"flag"
	"fmt"
	"github.com/open-bastion/open-bastion/internal/auth"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/egress"
	"github.com/open-bastion/open-bastion/internal/ingress"
	"github.com/open-bastion/open-bastion/internal/logs"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"strconv"
)

func main() {
	var configPath = flag.String("config-file", "", "(Optional) Specifies the configuration file path")

	flag.Parse()

	var sshServer ingress.Ingress
	var config config.Config
	var auth auth.Auth
	clientChannel := make(chan net.Conn)
	defer close(clientChannel)

	err := config.ParseConfig(*configPath)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = logs.Logger.InitLogger(config.EventsLogFile, config.SystemLogFile)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	logs.Logger.StartLogger()

	err = auth.ReadAuthorizedKeysFile(config.AuthorizedKeysFile)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = sshServer.ConfigSSHServer(auth.AuthorizedKeys, config.PrivateKeyFile)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = sshServer.ConfigTCPListener(config.ListenAddress + ":" + strconv.Itoa(config.ListenPort))

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	for i := 0; i < 5; i++ {
		go func(sshConfig *ssh.ServerConfig) {
			for {
				c := <-clientChannel

				handleTCPConn(&c, sshConfig)
			}
		}(sshServer.SSHServerConfig)
	}

	for {
		log.Println("Wait for a new connection")
		client, err := sshServer.TCPListener.Accept()

		if err != nil {
			log.Printf("failed to accept incoming connection: %v", err)
			continue
		}

		clientChannel <- client
	}
}

func handleTCPConn(c *net.Conn, sshConfig *ssh.ServerConfig) {
	// Before use, a handshake must be performed on the incoming
	// net.Conn.
	hsConn, chans, reqs, err := ssh.NewServerConn(*c, sshConfig)
	if err != nil {
		log.Print("Failed to handshake: ", err)
		return
	}

	log.Printf("User %s logged in with key %s", hsConn.User(), hsConn.Permissions.Extensions["pubkey-fp"])
	//logs.Logger.LogEvent("Logged in with key : " + string(hsConn.Permissions.Extensions["pubkey-fp"]) + "\n")

	go handleSSH(reqs, chans)
}

func handleSSH(reqs <-chan *ssh.Request, chans <-chan ssh.NewChannel) {
	// The incoming Request channel must be serviced.
	go func() {
		ssh.DiscardRequests(reqs)
	}()

	// Service the incoming Channel channel.
	for newChannel := range chans {
		log.Printf("Channel opened (type=%s)", newChannel.ChannelType())
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		//fmt.Print(string(newChannel.ExtraData()))

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
		}

		var payload string

		//The exec request should contains our backend information
		for req := range requests {
			if req.Type == "exec" {
				//The request payload is a raw byte array. Its 4 first bytes contain
				//its length so we need to remove them to correctly get the strings
				payload = string(req.Payload[4:])
				break
			}
		}

		if payload != "" {
			bc, err := egress.ParseBackendInfo(payload)

			if err != nil {
				log.Printf("Error : " + err.Error())
				return
			}

			// jump to new connection
			err = egress.DialSSH(channel, bc)

			if err != nil {
				fmt.Printf("Error : " + err.Error())
			}
		}
	}
}
