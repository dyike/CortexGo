package cortexgo

import (
	"fmt"
	"os"
)

// Logger handles logging functionality
type Logger struct {
	token string
}

func (l *Logger) configure() {
	l.token = os.Getenv("LOGFIRE_TOKEN")
}

func (l *Logger) debug(msg string) {
	if l.token != "" {
		fmt.Printf("[DEBUG] %s\n", msg)
	}
}

func (l *Logger) info(msg string) {
	if l.token != "" {
		fmt.Printf("[INFO] %s\n", msg)
	}
}

func (l *Logger) error(msg string) {
	if l.token != "" {
		fmt.Printf("[ERROR] %s\n", msg)
	}
}
