package main

import "time"

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

func (u *User) incrementMoves() {
	u.moves.number++
}

func (u *User) isMoveExceeded(b *Bot) bool {
	if (u.moves.number)/2 > 10 && time.Since(u.moves.sinceMove).Seconds() < 600 {
		b.exec(kickClient(u.clid, "Nie skacz po kanałach!"))
		delete(users, u.clid)
		return true
	}
	u.incrementMoves()
	return false
}
