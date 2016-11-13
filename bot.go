package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/overflow3d/ts3_/database"
)

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
		b.ID = "master"
		bots[b.ID] = b
		b.isMaster = true
		return nil
	}

	master, ok := bots["master"]
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
	if b.ID == "master" {
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
	cleanMaps := time.NewTicker(24 * time.Hour)
	for {
		select {
		case <-ping.C:
			b.writeToCon("version")
			infoLog.Println("Ping from bot: ", b.ID, " was send to telnet")
		case <-cleanMaps.C:
			log.Println("Clean maps")
		case <-b.stopPing:
			ping.Stop()
			cleanMaps.Stop()
			return
		}
	}
}

//Switcher for actions functions
func (b *Bot) notifyAction(r *Response) {
	switch r.action {
	case "notifytextmessage":

		cinfo, ok := usersByClid[r.params[0]["invokerid"]]
		if !ok {
			return
		}
		user, ok := users[cinfo]
		if !ok {
			return
		}
		if strings.Index(r.params[0]["msg"], "!") == 0 {
			b.actionMsg(r, user)
			return
		}
		debugLog.Println("Some normal message")
	case "notifyclientmoved":
		go func() {
			cinfo, ok := usersByClid[r.params[0]["clid"]]
			if !ok {
				return
			}
			user, ok := users[cinfo]
			if ok {
				user.Moves.MoveStatus++
				if user.Moves.MoveStatus == 2 {
					user.newRoomTrackerRecord(r.params[0]["ctid"])
					user.Moves.MoveStatus = 0
					if !user.IsAdmin {
						user.isMoveExceeded(b)
					}
				}
			}
		}()
	case "notifychanneledited":
		debugLog.Println(r.action)
	case "notifycliententerview":
		userDB, err := b.db.GetUser(r.params[0]["client_database_id"])
		if err != nil {
			errLog.Println(err)
		}
		if len(userDB) != 0 {
			retriveUser := &User{}
			retriveUser.unmarshalJSON(userDB)
			retriveUser.Clid = r.params[0]["clid"]
			users[retriveUser.Clidb] = retriveUser
			usersByClid[r.params[0]["clid"]] = retriveUser.Clidb
			if time.Since(retriveUser.Moves.SinceMove).Seconds() > 600 {
				retriveUser.Moves.Number = 0
			}
			if time.Since(retriveUser.BasicInfo.CreatedAT).Seconds() > 172800 {
				retriveUser.BasicInfo.IsRegistered = true
			}

		} else {
			if r.params[0]["client_database_id"] != "1" && r.params[0]["client_unique_identifier"] != "ServerQuery" {
				userS := newUser(r.params[0]["client_database_id"], r.params[0]["clid"], r.params[0]["client_nickname"])
				b.db.AddNewUser(r.params[0]["client_database_id"], userS)
				users[userS.Clidb] = userS
				usersByClid[userS.Clid] = userS.Clidb
			} else {
				warnLog.Println("Detected sever query bot")
			}
		}

	case "notifyclientleftview":
		userClid, ok := usersByClid[r.params[0]["clid"]]
		if !ok {
			warnLog.Println("Abnormal action")
			return
		}
		user, ok := users[userClid]
		if !ok {
			warnLog.Println("Abnormal action")
			return
		}
		if r.params[0]["reasonid"] == "5" {
			user.BasicInfo.Kick++
		}

		if r.params[0]["reasonid"] == "6" {
			user.BasicInfo.Ban++
		}

		if r.params[0]["reasonid"] == "4" {
			log.Println("Kick from channel")
			log.Println(r.params[0]["invokerid"], r.params[0]["invokername"])
		}

		b.db.AddNewUser(user.Clidb, user)
		delete(users, user.Clidb)
		delete(usersByClid, r.params[0]["clid"])
		debugLog.Println("Size of users map: ", len(users))
	case "notifychanneldescriptionchanged":
		//In case if I find function for it
		//Maybe if you are to lazy to add auto checking
		//for how much channel is empty, and you say user
		//to change date if they use room, otherwise edit
		return
	case "notifychannelcreated":
		return
	case "notifychannelmoved":
		return
	default:
		warnLog.Println("Unusual action: ", r.action)
		return
	}

}

func (b *Bot) actionMsg(r *Response, u *User) {
	switch {
	case strings.Index(r.params[0]["msg"], "!help") == 0:
		help := strings.SplitN(r.params[0]["msg"], " ", 2)
		if len(help) == 1 {
			go b.exec(sendMessage("1", r.params[0]["invokerid"], "Komendy, które możesz użyć"))
			return
		}

		debugLog.Println("Invoked help command")
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Zobaczysz to"))
		return
	case strings.Index(r.params[0]["msg"], "!uptime") == 0:
		bot := strings.SplitN(r.params[0]["msg"], " ", 2)
		if len(bot) == 2 {
			botCheck, ok := bots[bot[1]]
			if ok && b.ID == bot[1] {
				t := time.Unix(botCheck.Uptime, 0)
				since := time.Since(t)
				if since < 3600 {
					since.Minutes()
				}
				eventLog.Println("Bot ", botCheck.ID, " uptime is", since.String())
				return
			}

		}
		return

	case strings.Index(r.params[0]["msg"], "!quit") == 0:
		bot := strings.SplitN(r.params[0]["msg"], " ", 2)
		if len(bot) == 2 {
			botToClose, ok := bots[bot[1]]
			if ok {
				botToClose.conn.Close()
				warnLog.Println("User ", u.Nick, " invoked command to turn of bot")
				return
			}
			errLog.Println("There is no such bot as ", bot[1])
		}
		return

	case b.isMaster && strings.Index(r.params[0]["msg"], "!room") == 0:
		room := strings.SplitN(r.params[0]["msg"], " ", 3)
		if len(room) == 3 {
			pid, ok := isSpacer(room[1])
			if !ok {
				errLog.Println("No such spacer as ", room[1])
				return
			}
			go func() {
				defer func() { eventLog.Println("Room created with success: ", room[2]) }()
				cid := b.newRoom(room[2], pid, true, 0)
				if cid != "" {
					b.newRoom("", cid, false, 2)
				}
				client, err := b.exec(clientDBID(room[1], ""))
				if err != nil {
					errLog.Println(err)
					return
				}
				log.Println(client.params)
				_, errC := b.exec(setChannelAdmin(client.params[0]["cldbid"], cid))
				if errC != nil {
					errLog.Println(errC)
				}

			}()
		}

		return

	case b.isMaster && strings.Index(r.params[0]["msg"], "!create") == 0:
		if !u.IsAdmin {
			warnLog.Println("User ", u.Nick, " is not an Admin!")
			return
		}

		infoLog.Println("Created by: ", b.ID)
		newb := &Bot{}
		err := newb.newBot("teamspot.eu:10011", false)
		if err != nil {
			log.Println(err)
			return
		}

		infoLog.Println("New bot with id: ", newb.ID, "total of", len(bots), "in system", "bot created by ", u.Nick)
		newb.execAndIgnore(cmdsSub, true)

		return

	case b.isMaster && strings.Index(r.params[0]["msg"], "!admin") == 0:

		if !u.IsAdmin {
			warnLog.Println(u.Nick, " is not Admin")
			return
		}
		name := strings.SplitN(r.params[0]["msg"], " ", 2)
		if len(name) == 2 {
			go addAdmin(name[1], b)
		}

		return

	default:
		warnLog.Println("User invoked unknow command - ", u.Nick, " commad was ", r.params[0]["msg"])
	}

}

func (b *Bot) passNotify(notify string) {
	b.notify <- notify
}

func (b *Bot) passError(err string) {
	b.err <- err
}

//Mostly for pings, so you don't read
//Goroutine channels
func (b *Bot) writeToCon(s string) {
	fmt.Fprintf(b.conn, "%s\n\r", s)
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
