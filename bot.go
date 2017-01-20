package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
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
		eventLog.Println("Text message: ", r.params[0]["msg"])
	case "notifyclientmoved":
		//This if statment prevents from adding fakes moves to jump protection counter by admin move option
		if r.params[0]["invokername"] != "" || r.params[0]["invokerid"] != "" {
			eventLog.Println("Admin przeniósł użytkownika o id", r.params[0]["clid"], "na kanał", r.params[0]["ctid"])
			return
		}
		b.jumpProtection(r)
	case "notifychanneledited":
		//Add edition note in channel description
		//For more info who edited the channel
	case "notifycliententerview":
		userDB, err := b.db.GetRecord("users", r.params[0]["client_database_id"])
		if err != nil {
			errLog.Println(err)
		}

		if len(userDB) != 0 {
			retriveUser := &User{}
			retriveUser.unmarshalJSON(userDB)
			retriveUser.Clid = r.params[0]["clid"]
			retriveUser.Nick = r.params[0]["client_nickname"]
			users[retriveUser.Clidb] = retriveUser
			usersByClid[r.params[0]["clid"]] = retriveUser.Clidb
			if time.Since(retriveUser.Moves.SinceMove).Seconds() > 600 {
				retriveUser.Moves.Number = 0
			}

			if time.Since(retriveUser.BasicInfo.CreatedAT).Seconds() > 172800 {
				retriveUser.BasicInfo.IsRegistered = true
			}

			if retriveUser.BasicInfo.ReadRules == false {
				go func() {
					msg := customeMsg.RuleOne
					b.exec(sendMessage("1", retriveUser.Clid, msg))
					time.Sleep(2 * time.Nanosecond)
					msg = customeMsg.RuleTwo
					b.exec(sendMessage("1", retriveUser.Clid, msg))
				}()
			}
			if retriveUser.BasicInfo.IsPunished == true {
				go b.exec(clientMove(retriveUser.Clid, cfg.PunishRoom))
				go PunishRoom(b, retriveUser)
				retriveUser.BasicInfo.Punish.OriginTime = (retriveUser.BasicInfo.Punish.OriginTime - retriveUser.BasicInfo.Punish.CurrentTime) + (10 * float64(retriveUser.Moves.Warnings))
				timeLeft := retriveUser.BasicInfo.Punish.OriginTime - retriveUser.BasicInfo.Punish.CurrentTime
				strLeft := strconv.FormatFloat(timeLeft, 'f', 1, 64)
				go b.exec(clientMove(retriveUser.Clid, cfg.PunishRoom))
				go b.exec(clientPoke(retriveUser.Clid, "[color=red][b]Zostało Ci jeszcze "+strLeft+" sekund na karnym jeżyku, powodzenia :)[b][/color]"))
				eventLog.Println(retriveUser.Nick, "nie odbył pełnej kary na karnym jeżyku, zostało mu jeszcze", strLeft, "sekund")
			}

		} else {
			if r.params[0]["client_database_id"] != "1" && r.params[0]["client_unique_identifier"] != "ServerQuery" {
				userS := newUser(r.params[0]["client_database_id"], r.params[0]["clid"], r.params[0]["client_nickname"])
				b.db.AddRecord("users", r.params[0]["client_database_id"], userS)
				users[userS.Clidb] = userS
				usersByClid[userS.Clid] = userS.Clidb
				go func() {
					msg := customeMsg.RuleOne
					b.exec(sendMessage("1", r.params[0]["clid"], msg))
					time.Sleep(2 * time.Nanosecond)
					msg = customeMsg.RuleTwo
					b.exec(sendMessage("1", r.params[0]["clid"], msg))
				}()

			} else {
				warnLog.Println("Detected sever query bot")
			}
		}

	case "notifyclientleftview":
		if r.params[0]["reasonmsg"] == "" {
			go b.exec(kickClient(r.params[0]["invokerid"], "Wypełniaj powód kick/ban"))
		}
		if r.params[0]["invokeruid"] == "serveradmin" {
			return
		}
		if r.params[0]["reasonmsg"] == "deselected virtualserver" {
			infoLog.Println("Sever query bot event deselected virtualserver")
			return
		}
		userClid, ok := usersByClid[r.params[0]["clid"]]
		if !ok {
			if r.params[0]["reasonmsg"] == "connection lost" {
				debugLog.Println("Probably bot turning off", r.params)
				return
			}
			warnLog.Println("Abnormal action")
			return
		}
		user, ok := users[userClid]
		if !ok {
			warnLog.Println("Abnormal action")
			return
		}

		if r.params[0]["reasonid"] == "5" {
			infoLog.Println(user.Nick, "kicked from server by", r.params[0]["invokername"])
			user.BasicInfo.Kick++
			b.addKickBan("kicks", user.Clidb, r.params[0]["reasonmsg"], r.params[0]["invokername"])

		}

		if r.params[0]["reasonid"] == "6" {
			infoLog.Println(user.Nick, "banned from server by", r.params[0]["invokername"])
			user.BasicInfo.Ban++
			b.addKickBan("bans", user.Clidb, r.params[0]["reasonmsg"], r.params[0]["invokername"])
		}

		if r.params[0]["reasonid"] == "4" {
			infoLog.Println(user.Nick, " kicked from channel", r.params[0]["cid"])
		}

		b.db.AddRecord("users", user.Clidb, user)
		delete(users, user.Clidb)
		delete(usersByClid, r.params[0]["clid"])
	case "notifychanneldescriptionchanged":
		return
	case "notifychannelcreated":
		if r.params[0]["invokeruid"] != "serveradmin" {
			eventLog.Println("Room created manually by user:", r.params[0]["invokername"], "room name: ", r.params[0]["channel_name"])
			go b.roomFromNotify(r)
		} else {
			eventLog.Println("Room: ", r.params[0]["channel_name"], "created by bot", b.ID)
		}
		return
	case "notifychannelmoved":
		eventLog.Println("User", r.params[0]["invokername"], "moved channel ", r.params[0]["cid"], "to ", r.params[0]["cpid"])
		room, e := b.db.GetRecord("rooms", r.params[0]["cid"])
		if e != nil {
			errLog.Println("Database error:", e)
		}
		if len(room) != 0 {
			channel := &Channel{}
			channel.unmarshalJSON(room)
			channel.Spacer = r.params[0]["cpid"]
			err := b.db.AddRecord("rooms", channel.Cid, channel)
			if err != nil {
				errLog.Println("Database error:", e)
			}

		}
		return
	case "notifychanneldeleted":
		warnLog.Println("Room ", r.params[0]["cid"], "deleted by", r.params[0]["invokername"])
		delChannel := &DelChannel{
			Cid:        r.params[0]["cid"],
			DeletedBy:  r.params[0]["invokername"],
			InvokerUID: r.params[0]["invokeruid"],
			DeleteDate: time.Now(),
		}
		b.db.AddRecord("deletedRooms", delChannel.Cid, delChannel)
		_, err := b.db.GetRecord("rooms", r.params[0]["cid"])
		if err != nil {
			errLog.Println("No such room in databse cannot delete it from it")
			return
		}
		err = b.db.DeleteRecord("rooms", delChannel.Cid)
		if err != nil {
			errLog.Println(err)
		}
	default:
		warnLog.Println("Unusual action: ", r.action)
		return
	}

}

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

func (b *Bot) actionMsg(r *Response, u *User) {
	switch {
	case u.IsAdmin && strings.Index(r.params[0]["msg"], "!kicks") == 0:
		r.kicksHistoryCmd(u, b)
		return
	case u.IsAdmin && strings.Index(r.params[0]["msg"], "!kara") == 0:
		r.punishUserCmd(u, b)
		return
	case strings.Index(r.params[0]["msg"], "!help") == 0:
		r.helpCmd(u, b)
		return
	case strings.Index(r.params[0]["msg"], "!strefy") == 0:
		msg := customeMsg.Strefy
		go b.exec(sendMessage("1", r.params[0]["invokerid"], msg))
		return
	case strings.Index(r.params[0]["msg"], "!accept") == 0:
		r.acceptCmd(u, b)
		return
	case strings.Index(r.params[0]["msg"], "!uptime") == 0:
		r.upTimeCmd(u, b)
		return
	case u.IsAdmin && strings.Index(r.params[0]["msg"], "!quit") == 0:
		r.quitCmd(u, b)
		return
	case u.IsAdmin && b.isMaster && strings.Index(r.params[0]["msg"], "!room") == 0:
		r.createChannelCmd(u, b)
		return
	case u.IsAdmin && b.isMaster && strings.Index(r.params[0]["msg"], "!create") == 0:
		infoLog.Println("Created by: ", b.ID)
		r.createNewBotCmd(u, b)
		return
	case u.IsAdmin && strings.Index(r.params[0]["msg"], "!check") == 0:
		r.checkIfRoomOutOfDateCmd(u, b)
		return
	case strings.Index(r.params[0]["msg"], "!addMe") == 0:
		r.addUserAsAdminCmd(u, b)
		return
	case strings.Index(r.params[0]["msg"], "!setToken") == 0:
		r.setTokenCmd(u, b)
		return
	case strings.Index(r.params[0]["msg"], "!test") == 0:
		go b.exec(sendMessage("1", u.Clid, "Don't mind me I am just here for testing :)"))
		//registerUserAsPerm(b)
		return
	case strings.Index(r.params[0]["msg"], "!turnOffPoke") == 0:
		r.togglePokeCmd(u, b)
		return
	case strings.Index(r.params[0]["msg"], "!turnOffText") == 0:
		r.togglePrivateMsgCmd(u, b)
		return
	case u.IsAdmin && strings.Index(r.params[0]["msg"], "!debugUser") == 0:
		r.debugUser(u, b)
		return
	case strings.Index(r.params[0]["msg"], "!token") == 0:
		r.recoverChannelAdminCmd(u, b)
		return
	default:
		warnLog.Println("User invoked unknow command - ", u.Nick, " commad was ", r.params[0]["msg"])
		go b.exec(sendMessage("1", u.Clid, "Jeśli widzisz tą wiadomość prawdopodbnie wpisałeś złą komende albo nie masz do niej dostępu."))
	}

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
