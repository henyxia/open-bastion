package main

import (
	"context"
	"flag"
	"github.com/open-bastion/open-bastion/internal/auth"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/datastore"
	"github.com/open-bastion/open-bastion/internal/ingress"
	"github.com/open-bastion/open-bastion/internal/logger"
	"github.com/rs/zerolog/log"
	"strconv"
)

func main() {
	var configPath = flag.String("config-file", "", "(Optional) Specifies the configuration file path")

	flag.Parse()

	var sshServer ingress.Ingress
	var authInfo auth.Auth

	logger.InitDefaultLogger()

	bastionConfig, err := config.ParseConfig(*configPath)

	if err != nil {
		log.Fatal().Err(err).Msgf("error parsing configuration file")
	}

	logger.InitLogger(bastionConfig)
	ctx := logger.InitContextLogger(context.Background())
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

	sshServer.ListenAndServe(ctx, dataStore, bastionConfig)
}
