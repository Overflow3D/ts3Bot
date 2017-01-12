package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"
)

//User , ...
type User struct {
	Clidb     string
	Clid      string
	Nick      string
	Moves     *Moves
	BasicInfo *BasicInfo
	Spam      *Spam
	Perm      int
	IsAdmin   bool
}

//Moves , how much time user moved
type Moves struct {
	Number      int
	SinceMove   time.Time
	Warnings    int
	RoomTracker map[int64]string
	MoveStatus  int
}

//BasicInfo , user basic info
type BasicInfo struct {
	ReadRules    bool
	CreatedAT    time.Time
	LastSeen     time.Time
	IsRegistered bool
	Kick         int
	Ban          int
}

//Spam , user spam info
type Spam struct {
	TokenAttempts    int
	LastTokenAttempt time.Time
}

//KickAndBan , struct with kicks and bans of user
type KickAndBan struct {
	Invoker string
	Reason  string
}

var users = make(map[string]*User)
var usersByClid = make(map[string]string)

func newUser(dbID string, clid string, nick string) *User {
	newUser := &User{
		Clidb: dbID,
		Clid:  clid,
		Nick:  nick,
		Perm:  0,
		Moves: &Moves{
			Number:      0,
			SinceMove:   time.Now(),
			Warnings:    0,
			RoomTracker: make(map[int64]string),
			MoveStatus:  0,
		},
		BasicInfo: &BasicInfo{
			ReadRules:    false,
			CreatedAT:    time.Now(),
			LastSeen:     time.Now(),
			IsRegistered: false,
			Kick:         0,
			Ban:          0,
		},
		Spam: &Spam{
			TokenAttempts:    0,
			LastTokenAttempt: time.Now(),
		},
	}
	if dbID == cfg.HeadAdmin {
		eventLog.Println("HeadAdmin set to id", dbID, "with nickname ", newUser.Nick)
		newUser.IsAdmin = true
		newUser.Perm = 9999
	}
	return newUser
}

func (b *Bot) loadUsers() error {
	lists, err := b.exec(clientList())
	b.db.CreateBuckets("users")
	b.db.CreateSubBuckets("users", "kicks")
	b.db.CreateSubBuckets("users", "bans")
	var added, update int
	if err != nil {
		return err
	}

	for _, userTS := range lists.params {
		updateUser := new(User)
		if userTS["client_database_id"] != "1" {
			record, e := b.db.GetRecord("users", userTS["client_database_id"])
			if e != nil {
				errLog.Println("Database error: ", e)
			}
			if len(record) == 0 {
				added++
				user := newUser(userTS["client_database_id"], userTS["clid"], userTS["client_nickname"])
				users[userTS["client_database_id"]] = user
				usersByClid[userTS["clid"]] = userTS["client_database_id"]
				b.db.AddRecord("users", user.Clidb, user)
				continue
			}
			update++
			updateUser.unmarshalJSON(record)
			updateUser.Clid = userTS["clid"]
			users[userTS["client_database_id"]] = updateUser
			usersByClid[userTS["clid"]] = userTS["client_database_id"]

		}
	}

	debugLog.Println("Added", added, " and updated", update, "users on startup")
	return nil
}

func (u *User) incrementMoves() {
	u.Moves.Number++
}

func (u *User) isMoveExceeded(b *Bot) bool {
	if (u.Moves.Number) > 10 && time.Since(u.Moves.SinceMove).Seconds() < 600 {
		if u.Moves.Warnings >= 3 {
			log.Println("Ban time")
		}
		_, err := b.exec(kickClient(u.Clid, "Nie skacz po kanałach!"))
		if err != nil {
			errLog.Println(err)
		}
		u.Moves.Number = 0
		u.Moves.Warnings++
		b.db.AddRecord("users", u.Clidb, u)
		return true
	}
	u.incrementMoves()
	return false
}

func addAdmin(usr string, bot *Bot, headAdmin bool) {
	r, e := bot.exec(clientFind(usr))
	if e != nil {
		errLog.Println(e)
		return
	}
	userDB, ok := usersByClid[r.params[0]["clid"]]
	if !ok {
		return
	}
	user, ok := users[userDB]
	if ok {
		user.IsAdmin = true
		bot.db.AddRecord("users", user.Clidb, user)
		if headAdmin {
			user.Perm = 9999
			infoLog.Println("user", usr, "was set as an HeadAdmin")
		} else {
			infoLog.Println("user", usr, "was set as an Admin")
		}

	}

}

//Channel , channel struct
type Channel struct {
	Cid        string
	Spacer     string
	Name       string
	OwnerDB    string
	CreateDate time.Time
	CreatedBy  string
	Token      string
	Childs     []string
	Admins     []string
}

//DelChannel , delted channel struct
type DelChannel struct {
	Cid        string
	DeletedBy  string
	InvokerUID string
	DeleteDate time.Time
}

//Token , room tokens for id
type Token struct {
	Token      string
	Cid        string
	LastChange time.Time
	EditedBy   string
}

//getChannelList , allways to copy all existing rooms into channel struct
func (b *Bot) getChannelList() {
	var rooms int
	var skipped int
	err := b.db.CreateBuckets("rooms")
	b.db.CreateSubBuckets("rooms", "tokens")
	b.db.CreateBuckets("deletedRooms")
	if err != nil {
		debugLog.Println(err)
	}
	start := time.Now()
	defer func() {
		infoLog.Println("Loaded ", rooms, " and skipped ", skipped, "rooms in ", time.Since(start))
	}()
	infoLog.Println("Loading rooms")

	cl, err := b.exec(channelList())
	if err != nil {
		errLog.Println(err)
		return
	}
	spacers := []string{"595", "639"}
	for _, vMain := range cl.params {
		for _, spacer := range spacers {

			if vMain["pid"] == spacer {
				rooms++
				var admins []string
				adminList, err := b.exec(getChannelAdmin(vMain["cid"]))
				if err != nil {
				} else {
					//Return clidb
					for _, admin := range adminList.params {
						admins = append(admins, admin["cldbid"])
					}

				}

				var child []string

				for _, vSub := range cl.params {
					if vMain["cid"] == vSub["pid"] {
						child = append(child, vSub["cid"])
					}
				}

				tokens := randString(7)
				channel := &Channel{
					Cid:        vMain["cid"],
					Name:       vMain["channel_name"],
					Spacer:     spacer,
					CreateDate: time.Now(),
					Token:      tokens,
					CreatedBy:  b.ID,
					Admins:     admins,
					Childs:     child,
				}
				token := &Token{
					Token:      tokens,
					LastChange: time.Now(),
				}
				channel.Childs = child
				if len(channel.Admins) != 0 {
					//Shows channel admin for rooms
				}
				encode, err := json.Marshal(channel)
				if err != nil {
					log.Println(err)
				}
				r, e := b.db.GetRecord("rooms", channel.Cid)
				if e != nil {
					errLog.Println(e)
					continue
				}
				if len(r) == 0 {
					b.db.AddRecord("rooms", channel.Cid, encode)
					b.db.AddRecordSubBucket("rooms", "tokens", token.Token, token)
				} else {
					skipped++
				}

			}

		}

	}

}

func informOldUsers(users map[string]string) {
	bot := &Bot{}
	bot.newBot("teamspot.eu:10011", false)
	cmds := []*Command{
		useServer(cfg.ServerID),
		logIn(cfg.Login, cfg.Password),
		nickname("Reminder"),
		notifyRegister("channel", "0"),
	}
	bot.execAndIgnore(cmds, true)
}

func (b *Bot) newRoom(name string, pid string, isMain bool, subRooms int) ([]string, error) {
	var cids []string
	if !isMain {
		for i := 1; i <= subRooms; i++ {
			cid, err := b.exec(createRoom("Pokój "+strconv.Itoa(i), pid))
			if err != nil {
				errLog.Println(err)
			}
			cids = append(cids, cid.params[0]["cid"])
		}
		return cids, nil
	}

	cid, err := b.exec(createRoom(name, pid))
	if err != nil {
		return []string{}, err
	}
	infoLog.Println("Room with id: ", cid.params[0]["cid"], " was created")
	return []string{cid.params[0]["cid"]}, nil
}

func countUsers() int {
	return len(users)
}

func isSpacer(s string) (string, bool) {
	spacers := map[string]string{"test": "595", "V2": "26", "V3": "27", "S1": "19", "S2": "20", "S3": "21", "S4": "28"}
	for k, v := range spacers {
		if s == k {
			return v, true
		}
	}
	return "", false

}

func getDBFromClid(id string) string {
	dbID, ok := usersByClid[id]
	if !ok {
		return ""
	}
	return dbID
}

func (b *Bot) checkIfRoomOutDate() {
	start := time.Now()
	defer func() { debugLog.Println(time.Since(start)) }()
	bot := &Bot{}
	delCh := &DelChannel{}
	err := bot.newBot("teamspot.eu:10011", false)
	if err != nil {
		errLog.Println("Error while creatin bot")
	}
	bot.execAndIgnore(cmdsSub, true)
	r, e := bot.exec(channelList())
	if e != nil {
		errLog.Println(e)
		return
	}

	for i := range r.params {

		if isNormalUserArea(r.params[i]["pid"]) && r.params[i]["pid"] != "0" {
			room, isOut := bot.fetchChild(r.params[i]["pid"], r.params[i]["cid"], r.params)
			if isOut && len(room) >= 3 {
				debugLog.Println(room)
				bot.exec(deleteChannel(r.params[i]["cid"]))
				delCh.Cid = r.params[i]["cid"]
				delCh.DeletedBy = "Bot: " + bot.ID
				delCh.DeleteDate = time.Now()
				room, err := bot.db.GetRecord("rooms", delCh.Cid)
				if err != nil {
					errLog.Println("Database error: ", err)
					return
				}
				if len(room) != 0 {
					bot.db.DeleteRecord("rooms", delCh.Cid)
					debugLog.Println(room)
				}
				bot.db.AddRecord("deletedRooms", delCh.Cid, delCh)
				infoLog.Println("Room ", r.params[i]["channel_name"], "deleted by ", bot.ID)
			}
		}

	}
	infoLog.Println("Room cleaning done.")
	bot.conn.Close()
}

func (b *Bot) fetchChild(pid string, cid string, rooms []map[string]string) ([]string, bool) {
	var userRoom []string
	spacer := cfg.Spacers

	r, e := b.exec(channelInfo(cid))
	if e != nil {
		errLog.Println("Channel info error: ", e)
	}
	emptySince, err := strconv.Atoi(r.params[0]["seconds_empty"])
	if err != nil {
		return userRoom, false
	}
	if emptySince == -1 {
		return []string{"Channel in use"}, false
	}

	if emptySince < 1209600 {
		return []string{"Channel still below 7 days"}, false
	}
	userRoom = append(userRoom, r.params[0]["seconds_empty"])
	log.Println(r.params[0]["channel_name"])
	for _, v := range spacer {
		if pid == v {
			for _, room := range rooms {
				if room["pid"] == cid {
					cinfo, e := b.exec(channelInfo(room["cid"]))
					if e != nil {
						return userRoom, false
					}
					emptySinceChild, err := strconv.Atoi(cinfo.params[0]["seconds_empty"])
					if err != nil {
						continue
					}
					if emptySinceChild > 1209600 {
						userRoom = append(userRoom, cinfo.params[0]["seconds_empty"])
					}

				}
			}
		}
	}
	log.Println(userRoom)
	return userRoom, true
}

func (b *Bot) loadSpacers() []string {
	var spacers []string
	r, e := b.exec(channelList())
	if e != nil {
		errLog.Println(e)
	}

	for k := range r.params {

		if r.params[k]["pid"] == "0" {
			spacers = append(spacers, r.params[k]["cid"])
		}

	}
	return spacers
}

func isNormalUserArea(pcid string) bool {
	spacerPids := []string{"19", "20", "21", "28", "595"}
	for _, v := range spacerPids {
		if pcid == v {
			return true
		}
	}

	return false
}

func (u *User) checkTokeAttempts(valid []byte) (string, bool) {
	if len(valid) == 0 {
		if u.Spam.TokenAttempts < 3 {
			u.Spam.TokenAttempts++
			u.Spam.LastTokenAttempt = time.Now()
			return "Podany token nie istnieje w bazie danych, proszę spróbować ponownie", false
		}

		if u.Spam.TokenAttempts >= 3 && time.Since(u.Spam.LastTokenAttempt).Seconds() < 360 {
			return "Musisz odczekać 10 minut z powodu wprowadzenia 3 razy nieprawidłowego tokena", false
		}
		u.Spam.LastTokenAttempt = time.Now()
		u.Spam.TokenAttempts = 0

		return "Podany token nie istnieje w bazie danych, proszę spróbować ponownie", false
	}
	if u.Spam.TokenAttempts >= 3 && time.Since(u.Spam.LastTokenAttempt).Seconds() < 360 {
		return "Musisz odczekać 10 minut z powodu wprowadzenia 3 razy nieprawidłowego tokena", false
	}
	if u.Spam.TokenAttempts >= 3 && time.Since(u.Spam.LastTokenAttempt).Seconds() > 360 {
		u.Spam.LastTokenAttempt = time.Now()
		u.Spam.TokenAttempts = 0
	}

	return "Powinieneś odzyskać dostęp Channel Admina na swoim kanale, w razie problemów skontaktuj się z Administratorem.", true
}

func (b *Bot) addKickBan(bucket, clidb, reason, invoker string) {
	kicks, err := b.db.GetRecordSubBucket("users", bucket, clidb)
	kickBan := &KickAndBan{Invoker: invoker, Reason: reason}
	if err != nil {
		return
	}
	m := make(map[string][]*KickAndBan)
	time := time.Now()
	s := time.Format("02-01-2006")
	if len(kicks) == 0 {
		var kac []*KickAndBan
		kac = append(kac, kickBan)
		m[s] = kac
		b.db.AddRecordSubBucket("users", bucket, clidb, m)
		return
	}

	err = json.Unmarshal(kicks, &m)
	if err != nil {
		return
	}
	today, ok := m[s]
	if !ok {
		var kac []*KickAndBan
		kac = append(kac, kickBan)
		m[s] = kac
		b.db.AddRecordSubBucket("users", bucket, clidb, m)
		return
	}

	today = append(today, kickBan)
	m[s] = today
	b.db.AddRecordSubBucket("users", bucket, clidb, m)
	return
}

func (b *Bot) getUserKickBanHistory(bucket, clidb, date string) (string, error) {
	bans, err := b.db.GetRecordSubBucket("users", bucket, clidb)
	if err != nil {
		debugLog.Println(err)
		return "", err
	}
	if len(bans) == 0 {
		return "", errors.New("Nie kicków/banów dla danej osoby")
	}
	userKickBan := make(map[string][]*KickAndBan)
	err = json.Unmarshal(bans, &userKickBan)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	KickBan, ok := userKickBan[date]
	if !ok {
		return "", errors.New("Nie ma kicków/banów dla danego dnia")
	}
	for _, v := range KickBan {
		buffer.WriteString(" Od: " + v.Invoker + " powód " + v.Reason + " | ")
	}
	return buffer.String(), nil
}

func (u *User) unmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &u)
}

func (c *Channel) unmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &c)
}

func (t *Token) unmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &t)
}
