pacakge ssh



// Init set SSH server configuration
func Init() (error) {
	err = sshServer.ConfigSSHServer(auth.AuthorizedKeys, config.BastionConfig.PrivateKeyFile, dataStore)

	if err != nil {
		logger.Fatalf("error %v", err)
	}

	err = sshServer.ConfigTCPListener(config.BastionConfig.ListenAddress + ":" + strconv.Itoa(config.BastionConfig.ListenPort))

	if err != nil {
		logger.Fatalf("error %v", err)
	}


}

// InitOnly is false
func (s *LogService) InitOnly() (bool) {
	return false
}

func (s *LogService) Init() (error) {
	initLogger(config.BastionConfig)

	// cannot fail
	return nil
}

// Start does nothing
func (s *LogService) Start() (error) {
	return nil
}

// Stop does nothing
func (s *LogService) Stop() (error) {
	return nil
}
