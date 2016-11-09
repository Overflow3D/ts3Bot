package main

import (
	"log"
	"time"
)

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
	usersOnTeamSpeak := make(map[string]*User)
	if e != nil {
		log.Println(e)
	}
	var i int
	for _, list := range lists.params {
		if list["client_database_id"] != "1" && list["client_database_id"] != "" {
			usersOnTeamSpeak[list["client_database_id"]] = newUser(list["client_database_id"], list["clid"])
			i++
		}
	}
	log.Println("Loaded ", i, "users")
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

func countUsers() int {
	return len(users)
}
