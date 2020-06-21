package logger

// LogService represents the logger service
type LogService struct {
}

//LogConfigGetter represents the logger configuration
type LogConfigGetter interface {
	IsJSON() bool
	Level() int
	ReportCaller() bool
}

// Service manage the logger
var Service *LogService
