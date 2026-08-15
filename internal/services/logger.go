package services

import (
	"log"
	"os"
	"sync"
)

type CustomLogger struct {
	*log.Logger
}

var (
	Log  *CustomLogger
	once sync.Once
)

func init() {
	once.Do(func() {
		Log = &CustomLogger{
			Logger: log.New(os.Stdout, "APP: ", log.LstdFlags),
		}
	})
}

func (cl *CustomLogger) OriginAdvice(msg string) {
	cl.Printf(`Request from %s`, msg)
}

func (cl *CustomLogger) ErrorMessage(msg string) {
	cl.Printf(`Error %s`, msg)
}
