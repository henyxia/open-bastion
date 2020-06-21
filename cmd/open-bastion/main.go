package main

import (
	"log"
	"sync"

	"github.com/open-bastion/open-bastion/internal/service"
)

func main() {
    // wait group for all services to quit properly
    wg := sync.WaitGroup{}

    // start all services
	err := service.StartAll(&wg)
	if err != nil {
		log.Fatalln("unable to start open-bastion services: " + err.Error())
	}

    wg.Wait()
    log.Println("open-bastion successfully stopped")
}
