package main

import (
	"io"
	"log"
	"path"
	"runtime"
)

type Logger struct {
	*log.Logger
}

func NewLogger(out io.Writer, prefix string, flag int) *Logger {
	return &Logger{Logger: log.New(out, prefix, flag)}
}

func (self *Logger) Println(v ...interface{}) {
	self.PrintlnSkip(1, v...)
}

func (self *Logger) PrintlnSkip(skip int, v ...interface{}) {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		self.Logger.Println(v)
	} else {
		self.Logger.Println(path.Base(file), line, v)
	}
}

func (self *Logger) Panicln(v ...interface{}) {
	self.PaniclnSkip(1, v...)
}

func (self *Logger) PaniclnSkip(skip int, v ...interface{}) {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		self.Logger.Panicln(v)
	} else {
		self.Logger.Panicln(path.Base(file), line, v)
	}
}
