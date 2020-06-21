package service

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/logger"
)

var services []*Service

func startAndRegister(wg *sync.WaitGroup, service Service) {
	err := service.Init()
	if err != nil {
		log.Fatal().Msgf("error during initialization: %v", err)
	}

	if service.InitOnly() {
		return
	}

	go func() {
		wg.Add(1)
		defer wg.Done()

		services = append(services, &service)

		err = service.Start()
		if err != nil {
			log.Fatal().Msgf("unable to start service: %v", err)
		}
	}()
}

// StartAll starts all services
func StartAll(wg *sync.WaitGroup) (error) {
	wg.Add(1)
	defer wg.Done()

	startAndRegister(wg, config.BastionConfig)
	startAndRegister(wg, logger.Service)

	return nil
}
