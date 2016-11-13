package main

import (
	"encoding/json"
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

type BasicInfo struct {
	CreatedAT    time.Time
	LastSeen     time.Time
	IsRegistered bool
	Kick         int
	Ban          int
}

var users = make(map[string]*User)
var usersByClid = make(map[string]string)

func newUser(dbID string, clid string, nick string) *User {
	newUser := &User{
		Clidb: dbID,
		Clid:  clid,
		Nick:  nick,
		Moves: &Moves{
			Number:      0,
			SinceMove:   time.Now(),
			Warnings:    0,
			RoomTracker: make(map[int64]string),
			MoveStatus:  0,
		},
		BasicInfo: &BasicInfo{
			CreatedAT:    time.Now(),
			LastSeen:     time.Now(),
			IsRegistered: false,
			Kick:         0,
			Ban:          0,
		},
	}
	if dbID == cfg.HeadAdmin {
		eventLog.Println("HeadAdmin set to id", dbID, "with nickname ", newUser.Nick)
		newUser.IsAdmin = true
	}
	return newUser
}

func (b *Bot) loadUsers() error {
	lists, err := b.exec(clientList())
	var added int
	if err != nil {
		return err
	}

	for _, userTS := range lists.params {
		if userTS["client_database_id"] != "1" {
			added++
			user := newUser(userTS["client_database_id"], userTS["clid"], userTS["client_nickname"])
			users[userTS["client_database_id"]] = user
			usersByClid[userTS["clid"]] = userTS["client_database_id"]
			b.db.AddNewUser(user.Clidb, user)
		}
	}

	debugLog.Println("Added", added, "users on startup")
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
		b.db.AddNewUser(u.Clidb, u)
		return true
	}
	u.incrementMoves()
	return false
}

func addAdmin(usr string, bot *Bot) {
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
		bot.db.AddNewUser(user.Clidb, user)
		infoLog.Println("user", usr, "was set as an Admin")
	}

}

type Channel struct {
	Cid        string
	Spacer     string
	Name       string
	OwnerDB    string
	CreateDate time.Time
	CreatedBy  string
	Childs     []string
	Admins     []string
}

type DelChannel struct {
	Cid        string
	DeletedBy  string
	InvokerUID string
	DeleteDate time.Time
}

//getChannelList , allways to copy all existing rooms into channel struct
func (b *Bot) getChannelList() {
	var rooms int
	start := time.Now()
	defer func() {
		infoLog.Println("Loaded ", rooms, "rooms in ", time.Since(start))
	}()
	infoLog.Println("Loading rooms")

	channel := &Channel{}
	cl, err := b.exec(channelList())
	if err != nil {
		errLog.Println(err)
		return
	}
	spacers := []string{"595", "639"}
	for _, vMain := range cl.params {
		rooms++
		for _, spacer := range spacers {

			if vMain["pid"] == spacer {
				var admins []string
				channel.Spacer = spacer
				channel.Cid = vMain["cid"]
				channel.Name = vMain["channel_name"]
				channel.CreateDate = time.Now()
				adminList, err := b.exec(getChannelAdmin(vMain["cid"]))
				if err != nil {
					admins = []string{}
				} else {
					//Return clidb
					for _, admin := range adminList.params {
						admins = append(admins, admin["clidb"])
					}
				}
				var child []string

				for _, vSub := range cl.params {
					if vMain["cid"] == vSub["pid"] {
						child = append(child, vSub["cid"])
					}
				}

				channel.Childs = child
				encode, err := json.Marshal(channel)
				if err != nil {
					log.Println(err)
				}
				b.db.AddRoom([]byte(channel.Cid), encode)

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

func (u *User) unmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &u)
}

func (c *Channel) unmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &c)
}
