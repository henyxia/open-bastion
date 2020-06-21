package logger

import (
	"github.com/open-bastion/open-bastion/internal/config"
)

// InitOnly is true because the logger is only called through services after
func (s *LogService) InitOnly() (bool) {
	return true
}

// Init set logger configuration
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
