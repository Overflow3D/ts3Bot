package logger

import (
	"log"
	"os"
)

func Logs() {

}

func Error(e error) {
	err := log.New(os.Stderr, "\x1b[31;1m[ERROR]\x1b[0m ", log.LstdFlags|log.Lshortfile)
	err.Println(e)
}

func Debug(s string) {
	err := log.New(os.Stderr, "\x1b[36;1m[DEBUG]\x1b[0m ", log.LstdFlags|log.Lshortfile)
	err.Println(s)
}

func Info(s string) {
	err := log.New(os.Stderr, "\x1b[32;1m[INFO]\x1b[0m ", log.LstdFlags|log.Lshortfile)
	err.Println(s)
}
