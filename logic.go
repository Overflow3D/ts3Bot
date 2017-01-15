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
	IsPunished   bool
	Punish       *Punish
}

//Punish , punishment struct for user
type Punish struct {
	Multi       float64
	OriginTime  float64
	CurrentTime float64
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
			IsPunished:   false,
			Punish: &Punish{
				Multi:       0,
				OriginTime:  0,
				CurrentTime: 0,
			},
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

	for _, vMain := range cl.params {
		for _, spacer := range cfg.Spacer {

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

				channel := &Channel{
					Cid:        vMain["cid"],
					Name:       vMain["channel_name"],
					Spacer:     spacer,
					CreateDate: time.Now(),
					Token:      "",
					CreatedBy:  b.ID,
					Admins:     admins,
					Childs:     child,
				}
				channel.Childs = child
				if len(channel.Admins) != 0 {
					//Shows channel admin for rooms
				}

				r, e := b.db.GetRecord("rooms", channel.Cid)
				if e != nil {
					errLog.Println(e)
					continue
				}

				if len(r) == 0 {
					b.db.AddRecord("rooms", channel.Cid, channel)
				} else {
					skipped++
				}

			}

		}

	}

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
	for k, v := range cfg.Spacer {
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

func (b *Bot) checkIfRoomOutDate(remove bool, id string) {
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
	var roomsName bytes.Buffer
	roomCounter := 0
	for i := range r.params {

		if isNormalUserArea(r.params[i]["pid"]) && r.params[i]["pid"] != "0" {
			room, isOut := bot.fetchChild(r.params[i]["pid"], r.params[i]["cid"], r.params)
			if isOut && len(room) >= 3 {
				roomCounter++
				roomsName.WriteString(" " + r.params[i]["channel_name"] + " ")
				if remove {
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
					}
					bot.db.AddRecord("deletedRooms", delCh.Cid, delCh)
					infoLog.Println("Room ", r.params[i]["channel_name"], "deleted by ", bot.ID)
				}

			}
		}

	}
	if !remove {
		roomsName.WriteString(" " + strconv.Itoa(roomCounter) + " ")
		go b.exec(sendMessage("1", id, "Pokoje do usunięcia:"+roomsName.String()))
	}
	infoLog.Println("Room cleaning done.")
	bot.conn.Close()
}

func (b *Bot) fetchChild(pid string, cid string, rooms []map[string]string) ([]string, bool) {
	var userRoom []string
	var spacer []string
	for _, v := range cfg.Spacer {
		spacer = append(spacer, v)
	}
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

func isNormalUserArea(pcid string) bool {
	spacerPids := cfg.Spacer
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
	userInfo, e := b.exec(clientDBID(clidb, "-uid"))
	if e != nil {
		errLog.Println(e)
		return "", e
	}
	bans, err := b.db.GetRecordSubBucket("users", bucket, userInfo.params[0]["cldbid"])
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
	buffer.WriteString("\nWyniki dla podanej operacji to: ")
	for _, v := range KickBan {
		if v.Reason == "" {
			v.Reason = "Brak powodu"
		}
		buffer.WriteString("\n [color=green][b]Od:[/b][/color] " + v.Invoker + " [color=green][b]Powód:[/b][/color] " + v.Reason)
	}
	return buffer.String(), nil
}

func registerUserAsPerm(b *Bot) {
	for _, u := range users {
		if time.Since(u.BasicInfo.CreatedAT).Hours() > 280 && !u.BasicInfo.IsRegistered {
			_, e := b.exec(serverGroupAddClient(cfg.PermGroup, u.Clidb))
			if e != nil {
				errLog.Println("Error while adding to perm group", e)
				return
			}
			u.BasicInfo.IsRegistered = true
			b.db.AddRecord("users", u.Clidb, u)
			delete(users, u.Clidb)
			users[u.Clidb] = u
		}
	}
}

func PunishRoom(b *Bot, u *User) {
	time := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-time.C:

			res, e := b.exec(clientInfo(u.Clid))
			if e != nil {
				errLog.Println("error przy timerze użytkownika", u.Nick, "treść ", e)
				if e.Error() == "Error from telnet: 512 invalid clientID" {
					eventLog.Println("Użytkownik ", u.Nick, "opuścił teamspeak podczas odbywania kary")
					b.db.AddRecord("users", u.Clidb, u)
					return
				}
				continue
			}

			userRealTime, _ := users[u.Clidb]
			if userRealTime.BasicInfo.IsPunished == false {
				eventLog.Println("Kara przerwane przez administratora")
				u.BasicInfo.IsPunished = userRealTime.BasicInfo.IsPunished
				u.BasicInfo.Punish = &Punish{0, 0, 0}
				b.db.AddRecord("users", u.Clidb, u)
				go b.exec(clientMove(u.Clidb, cfg.GuestRoom))
				return
			}

			if u.BasicInfo.Punish.OriginTime < u.BasicInfo.Punish.CurrentTime {
				// Move user to Save room and break loop reset punish stats
				u.BasicInfo.IsPunished = false
				u.BasicInfo.Punish.CurrentTime = 0
				// delete(users, u.Clidb)
				// users[u.Clidb] = u
				b.db.AddRecord("users", u.Clidb, u)
				return
			}

			if res.params[0]["client_output_muted"] != "1" {
				u.BasicInfo.Punish.CurrentTime = u.BasicInfo.Punish.CurrentTime + 2
			}

		}
	}
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

//Przejrzeć kod dodatkow
func (b *Bot) jumpProtection(r *Response) {
	dbID, k := usersByClid[r.params[0]["clid"]]
	if !k {
		errLog.Println("Nie ma takiego użytkownika w pamięci pod clid", r.params[0]["clid"], "przenoszenie bota")
		return
	}
	u, ok := users[dbID]
	if !ok {
		errLog.Println("Nie ma takiego użytkownika w pamięci")
		return
	}
	if u.IsAdmin {
		return
	}
	if u.Moves.MoveStatus != 1 {
		u.Moves.MoveStatus++
		return
	}

	if u.Moves.Number == 0 {
		u.Moves.SinceMove = time.Now()
		u.Moves.Number++
		return
	}
	u.Moves.MoveStatus = 0

	sinceMove := time.Since(u.Moves.SinceMove).Seconds()
	if sinceMove > 360 {
		u.Moves.Number = 1
		u.Moves.SinceMove = time.Now()
		b.db.AddRecord("users", u.Clidb, u)
		return
	}
	if sinceMove < 60 {
		u.Moves.Number++
		u.Moves.SinceMove = time.Now()
		b.db.AddRecord("users", u.Clidb, u)
	}
	if sinceMove < 180 && u.Moves.Number >= 8 {
		if u.BasicInfo.IsPunished == false {
			punishTime := float64(180)
			if u.Moves.Warnings >= 3 {
				punishTime = 43200
				u.Moves.Warnings = 0
			}
			u.BasicInfo.IsPunished = true
			u.BasicInfo.Punish = &Punish{float64(u.Moves.Warnings + 1), punishTime, 0}
			go b.exec(clientMove(u.Clid, cfg.PunishRoom))
			go b.exec(clientPoke(u.Clid, "[color=red][b]180s kary za skakanie[/b][/color]"))
			go PunishRoom(b, u)

		}
		u.Moves.Number = 0
		u.Moves.Warnings++
		b.db.AddRecord("users", u.Clidb, u)
	}
}
