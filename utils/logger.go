// Package utils /
/*
MIT License

Copyright (c) 2023 Jonas Kaninda

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package utils

import (
	"fmt"
	"os"
	"time"
)

func Info(msg string, args ...any) {
	var currentTime = time.Now().Format("2006/01/02 15:04:05")
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s INFO: %s\n", currentTime, msg)
	} else {
		fmt.Printf("%s INFO: %s\n", currentTime, formattedMessage)
	}
}

// Warn warning message
func Warn(msg string, args ...any) {
	var currentTime = time.Now().Format("2006/01/02 15:04:05")
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s WARN: %s\n", currentTime, msg)
	} else {
		fmt.Printf("%s WARN: %s\n", currentTime, formattedMessage)
	}
}
func Error(msg string, args ...any) {
	var currentTime = time.Now().Format("2006/01/02 15:04:05")
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s ERROR: %s\n", currentTime, msg)
	} else {
		fmt.Printf("%s ERROR: %s\n", currentTime, formattedMessage)
	}
}

// Fatal logs an error message and exits the program
func Fatal(msg string, args ...any) {
	var currentTime = time.Now().Format("2006/01/02 15:04:05")
	// Fatal logs an error message and exits the program.
	formattedMessage := fmt.Sprintf(msg, args...)
	if len(args) == 0 {
		fmt.Printf("%s ERROR: %s\n", currentTime, msg)
		NotifyError(msg)
	} else {
		fmt.Printf("%s ERROR: %s\n", currentTime, formattedMessage)
		NotifyError(formattedMessage)

	}

	os.Exit(1)
}
