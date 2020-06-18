package main

import (
	"flag"
	"strconv"

	"github.com/open-bastion/open-bastion/internal/auth"
	obclient "github.com/open-bastion/open-bastion/internal/client"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/datastore"
	"github.com/open-bastion/open-bastion/internal/ingress"
	logger "github.com/open-bastion/open-bastion/internal/logger"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func main() {
	var configPath = flag.String("config-file", "", "(Optional) Specifies the configuration file path")

	flag.Parse()

	var sshServer ingress.Ingress
	var authInfo auth.Auth

	bastionConfig, err := config.ParseConfig(*configPath)

	if err != nil {
		log.Fatal().Err(err).Msgf("error parsing configuration file")
	}

	logger.InitLogger(bastionConfig)
	logger.Info("logger initialized")

	dataStore, err := datastore.InitStore(bastionConfig)

	if err != nil {
		logger.FatalfWithErr(err, "error")
	}
	logger.Infof("data store initialized, using: %v", dataStore.GetType())

	err = authInfo.ReadAuthorizedKeysFile(bastionConfig.AuthorizedKeysFile)

	if err != nil {
		logger.FatalfWithErr(err, "error reading authorized_keys file")
	}
	logger.Info("authorized_keys parsed")

	err = sshServer.ConfigSSHServer(authInfo.AuthorizedKeys, bastionConfig.PrivateKeyFile, dataStore)

	if err != nil {
		logger.FatalfWithErr(err, "error")
	}

	err = sshServer.ConfigTCPListener(bastionConfig.ListenAddress + ":" + strconv.Itoa(bastionConfig.ListenPort))

	if err != nil {
		logger.FatalfWithErr(err, "error")
	}
	logger.Info("server configured")

	listenAndServe(&sshServer, dataStore)
}

func listenAndServe(sshServer *ingress.Ingress, dataStore datastore.DataStore) {
	logger.Info("listening for new connections...")
	for {
		logger.Debug("waiting for a new connection...")
		client := new(obclient.Client)
		var err error

		client.TCPConnexion, err = sshServer.TCPListener.Accept()

		if err != nil {
			logger.WarnWithErr(err, "failed to handle the TCP connection")
			continue
		}

		go func(c *obclient.Client, sshConfig *ssh.ServerConfig, dataStore datastore.DataStore) {
			err := c.HandshakeSSH(sshConfig)

			if err != nil {
				logger.WarnWithErr(err, "failed to handshake")
				return
			}

			err = c.HandleSSHConnection()
			defer func() {
				if err := c.SshCommChan.Close(); err != nil {
					logger.WarnWithErr(err, "error closing the client communication channel")
				}

				if err := c.SSHConnexion.Close(); err != nil {
					logger.WarnWithErr(err, "error closing the SSH connection")
				}
			}()

			if err != nil {
				logger.WarnWithErr(err, "failed to handle the TCP connection")
				return
			}

			if c.BackendCommand == "bastion" {
				_ = c.RunCommand()
			} else {
				//The user has already been validated during the ssh handshake and should be good
				//We use the connecting user to parse its key
				c.SSHKey, err = dataStore.GetUserEgressPrivateKeySigner(client.User)

				if err != nil {
					_, _ = c.SshCommChan.Write([]byte("error accessing credentials"))

					logger.ErrorWithErr(err, "authenticated user could not access his eggress private key")
				}

				err = c.DialBackend()

				if err != nil {
					logger.WarnWithErr(err, "error dialing backend")
				}
			}
		}(client, sshServer.SSHServerConfig, dataStore)
	}
}
