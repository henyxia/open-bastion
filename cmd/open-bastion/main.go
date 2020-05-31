package main

import (
	"flag"
	"strconv"

	"github.com/open-bastion/open-bastion/internal/auth"
	"github.com/open-bastion/open-bastion/internal/client"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/ingress"
	logger "github.com/open-bastion/open-bastion/internal/logger"
	"github.com/open-bastion/open-bastion/internal/system"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func main() {
	var configPath = flag.String("config-file", "", "(Optional) Specifies the configuration file path")

	flag.Parse()

	var sshServer ingress.Ingress
	var auth auth.Auth
	clientChannel := make(chan *client.Client)
	defer close(clientChannel)

	err := config.BastionConfig.ParseConfig(*configPath)

	if err != nil {
		log.Fatal().Msgf("error parsing config %v", err)
	}

	logger.InitLogger(config.BastionConfig)

	dataStore, err := system.InitStore(config.BastionConfig)

	if err != nil {
		logger.Fatalf("error %v", err)
	}

	err = auth.ReadAuthorizedKeysFile(config.BastionConfig.AuthorizedKeysFile)

	if err != nil {
		logger.Fatalf("error %v", err)
	}

	err = sshServer.ConfigSSHServer(auth.AuthorizedKeys, config.BastionConfig.PrivateKeyFile, dataStore)

	if err != nil {
		logger.Fatalf("error %v", err)
	}

	err = sshServer.ConfigTCPListener(config.BastionConfig.ListenAddress + ":" + strconv.Itoa(config.BastionConfig.ListenPort))

	if err != nil {
		logger.Fatalf("error %v", err)
	}

	go func(sshConfig *ssh.ServerConfig) {
		for {
			c := <-clientChannel

			err = c.HandshakeSSH(sshConfig)

			if err != nil {
				logger.Warnf("Failed to handle the TCP conn: ", err)
				continue
			}

			err = c.HandleSSHConnexion()

			if err != nil {
				logger.Warnf("Failed to handle the TCP conn: ", err)
				continue
			}

			if c.BackendCommand == "bastion" {
				go c.RunCommand(dataStore)
			} else {
				go c.DialBackend()
			}
		}
	}(sshServer.SSHServerConfig)

	for {
		logger.Debug("Wait for a new connection")
		client := new(client.Client)

		client.TCPConnexion, err = sshServer.TCPListener.Accept()

		//Create new Client struct and pass it around

		if err != nil {
			logger.Warnf("failed to accept incoming connection: %v", err)
			continue
		}

		clientChannel <- client
	}
}
