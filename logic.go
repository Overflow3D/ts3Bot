package main

import (
	"log"
	"time"
)

//User , users struct
type User struct {
	clidb string
	clid  string
	moves *Moves
}

//Moves , how much time user moved
type Moves struct {
	number    int
	sinceMove time.Time
	warnings  int
}

var users = make(map[string]*User)

func addUser(dbID string, clid string) {
	_, ok := users[dbID]
	if !ok {
		users[dbID] = &User{
			clidb: dbID,
			clid:  clid,
			moves: &Moves{
				number:    0,
				sinceMove: time.Now(),
				warnings:  0,
			},
		}
	}
}

func newUser(dbID string, clid string) *User {
	return &User{
		clidb: dbID,
		clid:  clid,
		moves: &Moves{
			number:    0,
			sinceMove: time.Now(),
			warnings:  0,
		},
	}
}

func (b *Bot) loadUsers() {
	lists, e := b.exec(clientList())
	if e != nil {
		log.Println(e)
	}
	var i int
	for _, list := range lists.params {
		if list["client_database_id"] != "1" && list["client_database_id"] != "" {
			users[list["client_database_id"]] = newUser(list["client_database_id"], list["clid"])
			i++
		}
	}
	log.Println("Added ", i, "new users")
}

func (u *User) incrementMoves() {
	u.moves.number++
}

func (u *User) isMoveExceeded(b *Bot) bool {
	if (u.moves.number)/2 > 10 && time.Since(u.moves.sinceMove).Seconds() < 600 {
		b.exec(kickClient(u.clid, "Nie skacz po kanałach!"))
		u.moves.number = 0
		return true
	}
	u.incrementMoves()
	return false
}

func countUsers() int {
	return len(users)
}
