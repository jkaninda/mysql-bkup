package utils

import (
	"fmt"
	"os"
	"time"
)

var currentTime = time.Now().Format("2006/01/02 15:04:05")

func Info(msg string, args ...any) {
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s INFO: %s\n", currentTime, msg)
	} else {
		fmt.Printf("%s INFO: %s\n", currentTime, formattedMessage)
	}
}

// Warn warning message
func Warn(msg string, args ...any) {
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s WARN: %s\n", currentTime, msg)
	} else {
		fmt.Printf("%s WARN: %s\n", currentTime, formattedMessage)
	}
}
func Error(msg string, args ...any) {
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s ERROR: %s\n", currentTime, msg)
	} else {
		fmt.Printf("%s ERROR: %s\n", currentTime, formattedMessage)
	}
}
func Done(msg string, args ...any) {
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s INFO: %s\n", currentTime, msg)
	} else {
		fmt.Printf("%s INFO: %s\n", currentTime, formattedMessage)
	}
}

func Fatal(msg string, args ...any) {
	// Fatal logs an error message and exits the program.
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s ERROR: %s\n", currentTime, msg)
	} else {
		fmt.Printf("%s ERROR: %s\n", currentTime, formattedMessage)
	}
	os.Exit(1)
}
