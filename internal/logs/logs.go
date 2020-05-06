package logs

import (
	"errors"
	"time"
	"log"
	"os"
)

// Log structure
type Logs struct {
	logFile *os.File
	channel chan string
}

const (
	PanicLevel = 6
	FatalLevel = 5
	ErrorLevel = 4
	WarnLevel  = 3
	InfoLevel  = 2
	DebugLevel = 1
	TraceLevel = 0
)

// Logger need to be initialized and is used for all the logging operations through the program
var (
	System Logs
	User   Logs
)

// InitLogger open or create the log files and open the communication channels used by the logger
func (l *Logs) InitLogger(logPath string, async bool) (error) {
	var err error
	l.logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer l.logFile.Close()

	async = false

	if err != nil {
		return errors.New("Error opening or creating log file : " + err.Error())
	}

	if !async {
		return err
	}

	l.channel = make(chan string)

	go func() {
		for {
			log, ok := <-l.channel

			if !ok {
				break
			}

			l.logFile.Write([]byte(log))
		}
		close(l.channel)
	}()


	return err
}

func Fatal(message string) {
	log.Fatalln(message)
}

// LogEvent writes to the events log file
func (l *Logs) logGeneric(level int, msg string) error {
	var err error
	logMsg := time.Now().String() + " : " + msg + "\n"

	if l.channel != nil {
		l.channel <- logMsg
	} else {
		_, err = l.logFile.Write([]byte(logMsg))
	}

	return err
}

func (l *Logs) Panic(msg string) error { return l.logGeneric(PanicLevel, msg) }
func (l *Logs) Fatal(msg string) error { return l.logGeneric(FatalLevel, msg) }
func (l *Logs) Error(msg string) error { return l.logGeneric(ErrorLevel, msg) }
func (l *Logs) Warn(msg string) error { return l.logGeneric(WarnLevel, msg) }
func (l *Logs) Info(msg string) error { return l.logGeneric(InfoLevel, msg) }
func (l *Logs) Debug(msg string) error { return l.logGeneric(DebugLevel, msg) }
func (l *Logs) Trace(msg string) error { return l.logGeneric(TraceLevel, msg) }
