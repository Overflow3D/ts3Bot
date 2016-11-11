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
	Moves     *Moves
	BasicInfo *BasicInfo
	Perm      int
	IsAdmin   bool
}

//Moves , how much time user moved
type Moves struct {
	Number    int
	SinceMove time.Time
	Warnings  int
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

func newUser(dbID string, clid string) *User {
	newUser := &User{
		Clidb: dbID,
		Clid:  clid,
		Moves: &Moves{
			Number:    0,
			SinceMove: time.Now(),
			Warnings:  0,
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
		log.Println("HeadAdmin found in list")
		newUser.IsAdmin = true
	}
	return newUser
}

func (b *Bot) loadUsers() {
	lists, e := b.exec(clientList())
	if e != nil {
		log.Println(e)
	}

	for _, userTS := range lists.params {

		users[userTS["client_database_id"]] = newUser(userTS["client_database_id"], userTS["clid"])
		usersByClid[userTS["clid"]] = userTS["client_database_id"]

	}
	log.Println("Added", len(lists.params), "users")
}

func (u *User) incrementMoves() {
	u.Moves.Number++
}

func (u *User) isMoveExceeded(b *Bot) bool {
	if (u.Moves.Number)/2 > 10 && time.Since(u.Moves.SinceMove).Seconds() < 600 {
		b.exec(kickClient(u.Clid, "Nie skacz po kanałach!"))
		u.Moves.Number = 0
		u.Moves.Warnings++
		if u.Moves.Warnings >= 3 {
			// ban here
		}
		return true
	}
	u.incrementMoves()
	return false
}

func addAdmin(usr string, bot *Bot) {
	r, e := bot.exec(clientDBID(usr, ""))
	if e != nil {
		log.Println(e)
		return
	}

	user, ok := users[r.params[0]["cldbid"]]
	if ok {
		user.IsAdmin = true
		log.Println("user was set as an Admin")
	}

}

type Channel struct {
	Cid        string
	Spacer     string
	Name       string
	OwnerDB    string
	CreateDate time.Time
	Childs     []string
	Admins     []string
}

var channels map[string]*Channel

//getChannelList , allways to copy all existing rooms into channel struct
func (b *Bot) getChannelList() {
	chList := make(map[string]*Channel)
	start := time.Now()
	defer func() {
		log.Println("Loaded ", len(chList), "rooms in ", time.Since(start))
	}()
	log.Println("Loading rooms")

	channel := &Channel{}
	cl, err := b.exec(channelList())
	if err != nil {
		log.Println(err)
		return
	}
	spacers := []string{"595", "639"}
	for _, vMain := range cl.params {

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
				chList[vMain["cid"]] = channel
				encode, err := json.Marshal(channel)
				if err != nil {
					log.Println(err)
				}
				b.db.AddRoom([]byte(channel.Cid), encode)

			}

		}

	}

}

func (b *Bot) writeChannelsIntoMemo() {
	chList := make(map[string]*Channel)
	channel := &Channel{}
	chMap, err := b.db.ReadRooms()
	if err != nil {
		log.Fatalln(err)
	}
	for k, v := range chMap {
		err := channel.unmarshalJSON(v)
		if err != nil {
			log.Fatalln(err)
		}
		chList[k] = channel
	}
}

func (b *Bot) newRoom(name string, pid string, isMain bool, subRooms int) string {
	if !isMain {
		for i := 1; i <= subRooms; i++ {
			_, err := b.exec(createRoom("Pokój "+strconv.Itoa(i), pid))
			if err != nil {
				log.Println(err)
			}

		}
		return ""
	}

	cid, err := b.exec(createRoom(name, pid))
	if err != nil {
		log.Println(err)
	}
	log.Println(cid)
	return cid.params[0]["cid"]
}

func countUsers() int {
	return len(users)
}

func (u *User) unmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &u)
}

func (c *Channel) unmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &c)
}
