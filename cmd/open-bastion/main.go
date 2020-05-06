package main

import (
	"flag"
	"github.com/open-bastion/open-bastion/internal/auth"
	"github.com/open-bastion/open-bastion/internal/client"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/ingress"
	"github.com/open-bastion/open-bastion/internal/logs"
	"github.com/open-bastion/open-bastion/internal/system"
	"golang.org/x/crypto/ssh"
	"strconv"
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
		logs.Fatal("Error : " + err.Error())
	}

	// init system logger
	err = logs.System.InitLogger(config.BastionConfig.EventsLogFile, config.BastionConfig.AsyncEventsLog)

	if err != nil {
		logs.Fatal("Error : " + err.Error())
	}

	// init user logger
	err = logs.User.InitLogger(config.BastionConfig.SystemLogFile, config.BastionConfig.AsyncSystemLog)

	if err != nil {
		logs.System.Fatal("Error : " + err.Error())
	}

	dataStore, err := system.InitSystemStore(config.BastionConfig)

	if err != nil {
		logs.System.Fatal("Error : " + err.Error())
	}

	err = auth.ReadAuthorizedKeysFile(config.BastionConfig.AuthorizedKeysFile)

	if err != nil {
		logs.System.Fatal("Error : " + err.Error())
	}

	err = sshServer.ConfigSSHServer(auth.AuthorizedKeys, config.BastionConfig.PrivateKeyFile, dataStore)

	if err != nil {
		logs.System.Fatal("Error : " + err.Error())
	}

	err = sshServer.ConfigTCPListener(config.BastionConfig.ListenAddress + ":" + strconv.Itoa(config.BastionConfig.ListenPort))

	if err != nil {
		logs.System.Fatal("Error : " + err.Error())
	}

	go func(sshConfig *ssh.ServerConfig) {
		for {
			c := <-clientChannel

			err = c.HandshakeSSH(sshConfig)

			if err != nil {
				logs.System.Error("Failed to handle the TCP conn: "+err.Error())
				continue
			}

			err = c.HandleSSHConnexion()

			if err != nil {
				logs.System.Error("Failed to handle the TCP conn: "+err.Error())
				continue
			}

			if c.BackendCommand == "bastion" {
				c.RunCommand(dataStore)
			} else {
				go c.DialBackend()
			}
		}
	}(sshServer.SSHServerConfig)

	for {
		logs.System.Debug("Wait for a new connection")
		client := new(client.Client)

		client.TCPConnexion, err = sshServer.TCPListener.Accept()

		//Create new Client struct and pass it around

		if err != nil {
			logs.System.Warn("failed to accept incoming connection: "+err.Error())
			continue
		}

		clientChannel <- client
	}
}
