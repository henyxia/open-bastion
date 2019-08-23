package main

import (
	"fmt"
	"github.com/open-bastion/open-bastion/internal/config"
	"os"
)

func main() {
	args := os.Args

	config := new(config.Config)
	config.ParseConfig(args[1])

	fmt.Println(config.PermitPasswordLogin)
}
