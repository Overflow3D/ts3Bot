package main

import (
	"log"
	"os"
)

var (
	infoLog *log.Logger
	errLog  *log.Logger
	warnLog *log.Logger
)

func init() {
	flags := log.Lshortfile
	infoLog = log.New(os.Stdout, "\x1b[36;1m [ INFO ] \x1b[0m", flags)
	errLog = log.New(os.Stdout, "\x1b[31;1m [ ERROR ] \x1b[0m", flags)
	warnLog = log.New(os.Stdout, "\x1b[33;1m [ WARN ] \x1b[0m", flags)
}
