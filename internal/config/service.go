package config

import (
	"flag"

	"github.com/rs/zerolog/log"
)

// InitOnly is true here because we only need to read the configuration at the
// bastion start
func (conf config) InitOnly() bool {
	return true
}

// Init reads the configuration vars and file
func (conf config) Init() (error) {
	var configPath = flag.String("config-file", "", "(Optional) Specifies the configuration file path")

	err := conf.ParseConfig(*configPath)
	if err != nil {
		log.Fatal().Msgf("error parsing config %v", err)
	}

	return nil
}

// Start starts the config service
func (conf config) Start() (error) {
	return nil
}

func (conf config) Stop() (error) {
	return nil
}
