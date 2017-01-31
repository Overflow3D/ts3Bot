package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/overflow3d/ts3_/database"
)

const master = "master"

//Bot , is a bot struct
type Bot struct {
	ID       string
	conn     net.Conn
	output   chan string
	err      chan string
	notify   chan string
	stop     chan int
	stopPing chan struct{}
	isMaster bool
	resp     string
	db       database.Datastore
	Uptime   int64
}

//Response , represents telnet response
type Response struct {
	action string
	params []map[string]string
}

//TSerror , prase string errot into Error()
type TSerror struct {
	id  string
	msg string
}

func (e TSerror) Error() string {
	return fmt.Sprintf("Error from telnet: %s %s", e.id, e.msg)
}

var bots = make(map[string]*Bot)

//Creating new bot
func (b *Bot) newBot(addr string, isMaster bool) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		errLog.Println(err)
	}

	b.conn = conn

	scanCon := bufio.NewScanner(b.conn)
	scanCon.Split(scan)

	//Makes all bot channels
	b.makeChannels()

	//Launch goroutine for bot's connection scanner
	wg.Add(1)
	go b.scanCon(scanCon)

	//Adds separate notify goroutine
	go b.notifyRun()

	//Launch goroutine to fetch telnet response
	go b.run()

	//Launches ping
	go b.botSchedules()

	b.Uptime = time.Now().Unix()
	if isMaster {
		b.ID = master
		bots[b.ID] = b
		b.isMaster = true
		return nil
	}

	master, ok := bots[master]
	if !ok {
		return errors.New("Couldn't copy database")
	}

	b.ID = randString(5)
	bots[b.ID] = b
	b.db = master.db
	return nil

}

func (b *Bot) scanCon(s *bufio.Scanner) {
	defer b.cleanUp()

	for {
		s.Scan()
		b.output <- s.Text()
		//Checks if connection is openned or any other error
		e := s.Err()
		if e != nil {
			return
		}
	}
}

func (b *Bot) makeChannels() {
	b.output = make(chan string)
	b.err = make(chan string)
	b.notify = make(chan string)
	b.stop = make(chan int)
	b.stopPing = make(chan struct{})
}

//Cleans up after closing bot
func (b *Bot) cleanUp() {
	b.stop <- 1
	close(b.output)
	close(b.stopPing)
	if b.ID == master {
		for bot := range bots {
			i := bots[bot]
			i.conn.Close()
		}
	}
	warnLog.Println("Bot", b.ID, "stopped his work.")
	wg.Done()
}

func (b *Bot) run() {
	defer func() {
		warnLog.Println("Bot's", b.ID, "fetching stopped due to bot turning off")
	}()
	for {
		select {
		case m, ok := <-b.output:
			if ok {
				if strings.Index(m, "TS3") == 0 || strings.Index(m, "Welcome") == 0 || strings.Index(m, "version") == 0 {
					continue
				}
				if strings.Index(m, "error") == 0 {
					b.passError(m)
					continue
				} else if strings.Index(m, "notify") == 0 {
					b.passNotify(m)
					continue
				} else {
					b.resp = m
				}
			}

		case <-b.stop:
			return
		}

	}
}

func (b *Bot) notifyRun() {
	for {
		notificatio := <-b.notify
		r := formatResponse(notificatio, "notify")
		b.notifyAction(r)
	}
}

//Schedules for bot
func (b *Bot) botSchedules() {
	ping := time.NewTicker(305 * time.Second)
	cleanRooms := time.NewTicker(48 * time.Hour)
	registered := time.NewTicker(5 * time.Hour)
	pointsT := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ping.C:
			go b.exec(version())
			infoLog.Println("Ping from bot: ", b.ID, " was send to telnet")
		case <-cleanRooms.C:
			if b.ID == master {
				b.checkIfRoomOutDate(true, "0")
				infoLog.Println("Check for empty rooms")
			}
		case <-registered.C:
			registerUserAsPerm(b)
			eventLog.Println("Sprawdzanie czy użytkownik spełnia wymagania rangi zarejestrowany")
		case <-pointsT.C:
			b.givePoints()
			eventLog.Println("Users gets thier points for being active on teamspeak")
		case <-b.stopPing:
			ping.Stop()
			cleanRooms.Stop()
			return
		}
	}
}

//Switcher for actions functions

func (b *Bot) roomFromNotify(r *Response) {
	debugLog.Println(r.params)
	channel := &Channel{}
	encodedRoom, err := b.db.GetRecord("rooms", r.params[0]["cid"])
	if err != nil {
		errLog.Println("Database error: ", err)
	}

	for _, s := range cfg.Spacer {
		if r.params[0]["cpid"] == s {
			if len(encodedRoom) == 0 {
				owner, er := b.exec(clientFind(r.params[0]["channel_name"]))
				if er != nil {
					errLog.Println("Incorrect owner id:", err)
					b.exec(sendMessage("1", r.params[0]["invokerid"], "Wprowadziłeś niepoprawną nazwę właściciela kanału wyśli użytkownikowi w prywatnej wiadomości token, który otrzymałeś pod spodem. Błąd telnet: "+er.Error()))

				}
				clientDB := getDBFromClid(owner.params[0]["clid"])
				if clientDB != "" {
					b.exec(setChannelAdmin(clientDB, r.params[0]["cid"]))
					channel.Admins = []string{clientDB}
				} else {
					channel.Admins = []string{}
				}
				token := randString(7)
				tok := &Token{Token: token, Cid: r.params[0]["cid"]}
				infoLog.Println("Creating main room")
				channel.Cid = r.params[0]["cid"]
				channel.Spacer = r.params[0]["cpid"]
				channel.OwnerDB = clientDB
				channel.CreatedBy = r.params[0]["invokername"]
				channel.CreateDate = time.Now()
				channel.Name = r.params[0]["channel_name"]
				channel.Token = token
				channel.Childs = []string{}

				errr := b.db.AddRecord("rooms", channel.Cid, channel)
				if errr != nil {
					errLog.Println(err)
				}
				b.db.AddRecordSubBucket("rooms", "tokens", token, tok)
				go b.exec(sendMessage("1", r.params[0]["invokerid"], "Token dla utworzonego pokoju to: [b][color=red]"+tok.Token+"[/color][/b]"))
				go b.exec(sendMessage("1", owner.params[0]["clid"], "Token dla Twojego kanału by odzyskać channel Admina to [b][color=red] "+tok.Token+" [/color][/b]"))
				return
			}
			err = channel.unmarshalJSON(encodedRoom)
			if err != nil {
				errLog.Println("Channel decoding error:", err)
			}
			channel.Childs = append(channel.Childs, r.params[0]["cid"])
			b.db.AddRecord("rooms", channel.Cid, channel)
			return
		}
	}
	eventLog.Println("Manually creating an extra room for", r.params[0]["cpid"])
}

func (b *Bot) passNotify(notify string) {
	b.notify <- notify
}

func (b *Bot) passError(err string) {
	b.err <- err
}

//Formats output from telnet into Reponse struct
func formatResponse(s string, action string) *Response {
	r := &Response{}

	var splitResponse []string
	if action == "cmd" {
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

//Converts telnet error string to error struct
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
	if e.id != "0" && e.id != "" {
		return e
	}
	return nil
}

//Scans telnet output from connection
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
