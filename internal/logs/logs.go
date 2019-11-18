package logs

import (
	"errors"
	"os"
	"time"
)

// Logs contains the files pointers and channels used by the logging system
type Logs struct {
	eventsLogFile *os.File
	systemLogFile *os.File
	eventsChannel chan string
	systemChannel chan string
}

// Logger need to be initialized and is used for all the logging operations through the program
var Logger Logs

// InitLogger open or create the log files and open the communication channels used by the logger
func (l *Logs) InitLogger(eventsLogFilePath string, systemLogFilePath string) error {
	var err error
	l.eventsLogFile, err = os.OpenFile(eventsLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return errors.New("Error opening or creating events log file : " + err.Error())
	}

	l.systemLogFile, err = os.OpenFile(systemLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return errors.New("Error opening or creating system log file : " + err.Error())
	}

	l.eventsChannel = make(chan string)
	l.systemChannel = make(chan string)

	return nil
}

// StartLogger starts the goroutines used to communicate with the logger
func (l *Logs) StartLogger() {
	go func() {
		for {
			log, ok := <-l.eventsChannel

			if !ok {
				break
			}

			l.eventsLogFile.Write([]byte(log))
		}
	}()

	go func() {
		for {
			log, ok := <-l.systemChannel

			if !ok {
				break
			}

			l.systemLogFile.Write([]byte(log))
		}
	}()
}

// StopLogger clean up the allocated ressources
func (l *Logs) StopLogger() {
	close(l.eventsChannel)
	close(l.systemChannel)

	l.eventsLogFile.Close()
	l.systemLogFile.Close()
}

// LogEvent writes to the events log file
func (l *Logs) LogEvent(msg string) {
	logMsg := time.Now().String() + " : " + msg + "\n"
	l.eventsChannel <- logMsg
}

// LogSystem writes to the system log file
func (l *Logs) LogSystem(msg string) {
	logMsg := time.Now().String() + " : " + msg + "\n"
	l.systemChannel <- logMsg
}

//TODO sync log
