package main

import (
	"bufio"
	"bytes"
	"log"
	"net"
)

//Bot , is a bot struct
type Bot struct {
	ID     string
	conn   net.Conn
	output chan string
	notify chan string
	stop   chan int
}

var bots = make(map[string]*Bot)

func newBot(addr string) {
	bot := new(Bot)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
	}

	bot.conn = conn

	scanCon := bufio.NewScanner(bot.conn)
	scanCon.Split(scan)
	bot.output = make(chan string)
	bot.notify = make(chan string)
	bot.stop = make(chan int)

	//Launch goroutine for bot's connection scanner
	wg.Add(1)
	go bot.scanCon(scanCon)

	//Launch goroutine to fetch telnet response
	go bot.run()

}

func (b *Bot) scanCon(s *bufio.Scanner) {
	defer func() {
		log.Println("Bot: ", b.ID, "stopped his work.")
		b.stop <- 1
		wg.Done()
	}()

	for {
		s.Scan()
		b.output <- s.Text()
	}
}

func (b *Bot) run() {
	defer func() {
		log.Println("Bot's", b.ID, "fetching stopped due to bot turning off")
	}()
	for {
		select {
		case m, ok := <-b.output:
			if ok {
				log.Println(m)
				if m == "TS3" {
					go b.passNotify(m)
				}
			}
		case m, ok := <-b.notify:
			if ok {
				log.Println("Notify recognized in message: ", m)
			}
		case <-b.stop:
			return
		}

	}
}

func (b *Bot) passNotify(notify string) {
	b.notify <- notify
}

func scan(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\r")); i >= 0 {
		return i + 2, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
