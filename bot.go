package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"
)

//Bot , is a bot struct
type Bot struct {
	ID     string
	conn   net.Conn
	output chan string
	err    chan string
	notify chan string
	stop   chan int
	resp   string
}

//Response , represents telnet response
type Response struct {
	action string
	params []map[string]string
}

type TSerror struct {
	id  string
	msg string
}

func (e TSerror) Error() string {
	return fmt.Sprintf("Error from telnet: %s %s", e.id, e.msg)
}

var bots = make(map[string]*Bot)

func newBot(addr string, isMaster bool) {
	bot := new(Bot)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
	}

	bot.conn = conn

	scanCon := bufio.NewScanner(bot.conn)
	scanCon.Split(scan)
	bot.output = make(chan string)
	bot.err = make(chan string)
	bot.notify = make(chan string)
	bot.stop = make(chan int)

	//Launch goroutine for bot's connection scanner
	wg.Add(1)
	go bot.scanCon(scanCon)

	//Launch goroutine to fetch telnet response
	go bot.run()

	if isMaster {
		bots["master"] = bot
		return
	}

	bots["xxx"] = bot

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
				if strings.Index(m, "TS3") == 0 || strings.Index(m, "Welcome") == 0 || strings.Index(m, "version") == 0 {
					continue
				}

				if strings.Index(m, "error") == 0 {
					go b.passError(m)
					continue
				}

				if strings.Index(m, "notify") == 0 {
					go b.passNotify(m)
					continue
				} else {
					b.resp = m
				}
			}
			//case notify
		case x, ok := <-b.notify:
			if ok {
				log.Println(x)
			}

		case <-b.stop:
			return
		}

	}
}

func (b *Bot) passNotify(notify string) {
	b.notify <- notify
}

func (b *Bot) passError(err string) {
	b.err <- err
}

func (b *Bot) writeToCon(s string) {
	fmt.Fprintf(b.conn, "%s\n\r", s)
	err := <-b.err
	res := b.resp
	if res == "" {
		return
	}

	log.Println(formatResponse(res, "c"))
	log.Println(formatError(err))
}

func formatResponse(s string, action string) *Response {
	r := &Response{}

	var splitResponse []string
	if action == "c" {
		r.action = "Cmd_Response"
		splitResponse = strings.Split(s, "|")
	} else {
		notifystr := strings.SplitN(s, " ", 2)
		r.action = notifystr[0]
		splitResponse = strings.Split(notifystr[1], "|")

	}
	for i := range splitResponse {
		r.params = append(r.params, make(map[string]string))
		splitWhiteSpaces := strings.Split(splitResponse[i], " ")

		for j := range splitWhiteSpaces {

			splitParams := strings.SplitN(splitWhiteSpaces[j], "=", 2)
			if len(splitParams) > 1 {
				r.params[i][splitParams[0]] = unescape(splitParams[1])
			} else {
				r.params[i][splitParams[0]] = ""
			}
		}

	}

	return r
}

func formatError(s string) error {
	e := &TSerror{}
	errorSplit := strings.Split(s, " ")
	for i := range errorSplit {

		eParams := strings.SplitN(errorSplit[i], "=", 2)
		if len(eParams) > 1 {
			if eParams[0] == "id" {
				e.id = eParams[1]
			} else if eParams[0] == "msg" {
				e.msg = unescape(eParams[1])
			}
		} else {
			continue
		}

	}
	if e.id != "0" && e.id == "" {
		return e
	}

	return nil
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
