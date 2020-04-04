package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/open-bastion/open-bastion/internal/auth"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/egress"
	"github.com/open-bastion/open-bastion/internal/ingress"
	"github.com/open-bastion/open-bastion/internal/logs"
	"github.com/open-bastion/open-bastion/internal/system"
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
	var auth auth.Auth
	clientChannel := make(chan *Client)
	defer close(clientChannel)

	err := config.BastionConfig.ParseConfig(*configPath)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = logs.Logger.InitLogger(config.BastionConfig.EventsLogFile, config.BastionConfig.SystemLogFile, config.BastionConfig.AsyncEventsLog, config.BastionConfig.AsyncSystemLog)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	logs.Logger.StartLogger()
	defer logs.Logger.StopLogger()

	dataStore, err := system.InitSystemStore()

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = auth.ReadAuthorizedKeysFile(config.BastionConfig.AuthorizedKeysFile)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = sshServer.ConfigSSHServer(auth.AuthorizedKeys, config.BastionConfig.PrivateKeyFile, dataStore)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = sshServer.ConfigTCPListener(config.BastionConfig.ListenAddress + ":" + strconv.Itoa(config.BastionConfig.ListenPort))

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	go func(sshConfig *ssh.ServerConfig) {
		for {
			c := <-clientChannel

			err = c.handshakeSSH(sshConfig)

			if err != nil {
				log.Print("Failed to handle the TCP conn: ", err)
				continue
			}

			err = c.handleSSHConnexion()

			if err != nil {
				log.Print("Failed to handle the TCP conn: ", err)
				continue
			}

			if c.BackendCommand == "bastion" {
				go c.runCommand(dataStore)
			} else {
				go c.dialBackend()
			}
		}
	}(sshServer.SSHServerConfig)

	for {
		log.Println("Wait for a new connection")
		client := new(Client)

		client.TCPConnexion, err = sshServer.TCPListener.Accept()

		//Create new Client struct and pass it around

		if err != nil {
			log.Printf("failed to accept incoming connection: %v", err)
			continue
		}

		clientChannel <- client
	}
}
