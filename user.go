package main

import "time"

//User , ...
type User struct {
	Clidb     string
	Clid      string
	Nick      string
	Moves     *Moves
	BasicInfo *BasicInfo
	Spam      *Spam
	Punish    *Punish
	Points    *Points
	IsAdmin   bool
	Admin     *Admin
}

//Admin , information about admins righs
type Admin struct {
	Perms int
}

//Moves , how much time user moved
type Moves struct {
	Number     int
	SinceMove  time.Time
	Warnings   int
	MoveStatus int
}

//BasicInfo , user basic info
type BasicInfo struct {
	ReadRules    bool
	CreatedAT    time.Time
	LastSeen     time.Time
	IsRegistered bool
}

//Punish , punishment struct for user
type Punish struct {
	IsPunished  bool
	Multi       float64
	OriginTime  float64
	CurrentTime float64
}

//Spam , user spam info
type Spam struct {
	TokenAttempts    int
	LastTokenAttempt time.Time
}

type Points struct {
	Amount int
}

func createUser(clientDB, clientID, nick string) *User {
	user := &User{
		Clidb:     clientDB,
		Clid:      clientID,
		Nick:      nick,
		Moves:     createUserMoves(),
		BasicInfo: createUserBaiscInfo(),
		Punish:    createUserPunisInfo(),
		Spam:      createUserSpamInfo(),
		Points:    createUserPointsInfo(),
		IsAdmin:   false,
		Admin:     createUserAdminInfo(),
	}
	if clientDB == cfg.HeadAdmin {
		user.setAdmin()
	}
	return user
}

func createUserMoves() *Moves {
	return &Moves{0, time.Now(), 0, 0}
}

func createUserBaiscInfo() *BasicInfo {
	return &BasicInfo{false, time.Now(), time.Now(), false}
}

func createUserPunisInfo() *Punish {
	return &Punish{false, 0, 0, 0}
}

func createUserPointsInfo() *Points {
	return &Points{0}
}

func createUserSpamInfo() *Spam {
	return &Spam{0, time.Now()}
}

func createUserAdminInfo() *Admin {
	return &Admin{0}
}

func (u *User) setAdmin() {
	eventLog.Println("HeadAdmin set to id", u.Clidb, "with nickname ", u.Nick)
	u.IsAdmin = true
	u.Admin.Perms = 9999
}

func (m *Moves) incrementMoves() {
	m.Number++
	m.SinceMove = time.Now()
}

func (m *Moves) incrementWarnings() {
	m.Warnings++
}

func (p *Points) incrementPoints(points int) {
	p.Amount = p.Amount + points
}
