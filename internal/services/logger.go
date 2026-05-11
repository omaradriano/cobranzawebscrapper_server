package services

import (
	"log"
	"os"
)

type CustomLogger struct {
	*log.Logger // Embebemos el logger estándar
}

func NewLogger() *CustomLogger {
	return &CustomLogger{
		Logger: log.New(os.Stdout, "APP: ", log.LstdFlags),
	}
}

// Definimos el método con Pointer Receiver (*)
func (cl *CustomLogger) OriginAdvice(msg string) {
	cl.Printf(`Request from %s`, msg)
	// cl.Println(`----------------------------------------`)
}

func (cl *CustomLogger) ErrorMessage(msg string) {
	cl.Printf(`Error %s`, msg)
	// cl.Println(`----------------------------------------`)
}
