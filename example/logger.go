package main

import (
	"fmt"
	"os"

	"github.com/zeebox/go-http-middleware"
)

type FileLogger struct {
	File *os.File
}

func NewFileLogger(p string) (f FileLogger, err error) {
	f.File, err = os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)

	return
}

func (f FileLogger) Log(l middleware.LogEntry) {
	logStr := fmt.Sprintf("%+v\n", l)

	f.File.WriteString(logStr)
}
