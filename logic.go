package main

import (
	"encoding/json"
	"log"
	"strconv"
	"time"
)

//User , ...
type User struct {
	Clidb   string
	Clid    string
	Moves   *Moves
	Perm    int
	IsAdmin bool
}

//Moves , how much time user moved
type Moves struct {
	Number    int
	SinceMove time.Time
	Warnings  int
}

var users = make(map[string]*User)

func addUser(dbID string, clid string) {
	_, ok := users[dbID]
	if !ok {
		users[dbID] = &User{
			Clidb: dbID,
			Clid:  clid,
			Moves: &Moves{
				Number:    0,
				SinceMove: time.Now(),
				Warnings:  0,
			},
		}
	}
}

func newUser(dbID string, clid string) *User {
	return &User{
		Clidb: dbID,
		Clid:  clid,
		Moves: &Moves{
			Number:    0,
			SinceMove: time.Now(),
			Warnings:  0,
		},
	}
}

func (b *Bot) loadUsers() {
	lists, e := b.exec(clientList())
	// usersOnTeamSpeak := make(map[string]*User)
	if e != nil {
		log.Println(e)
	}
	userList := b.db.LoadUserFromDB()
	var updateUser int
	var addedUser int
	for _, userTS := range lists.params {
		if userTS["client_database_id"] != "1" && userTS["client_database_id"] != "" {
			for key, userDB := range userList {
				if key == userTS["client_database_id"] {
					user := &User{}
					err := user.unmarshalJSON(userDB)
					if err != nil {
						log.Println("error")
					}
					user.Clid = userTS["clid"]
					users[user.Clidb] = user
					updateUser++
				} else {
					users[userTS["client_database_id"]] = newUser(userTS["client_database_id"], userTS["clid"])
					//b.db.AddNewUser(userTS["client_database_id"], userTS["clid"])
					addedUser++
				}

			}

		}
	}
	log.Println("Updated ", updateUser, "and added", addedUser, "users")
	log.Println("wielkosc mapy:", countUsers())
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

			}

		}

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
