package main

import (
	"bytes"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
)

//actionMsg , seek proper command to execude
func (b *Bot) actionMsg(r *Response, u *User) {
	indexOfWhiteSpace := strings.Index(r.params[0]["msg"], " ")
	if indexOfWhiteSpace == -1 {
		indexOfWhiteSpace = len(r.params[0]["msg"])
	}
	commandKey := strings.ToLower(r.params[0]["msg"][:indexOfWhiteSpace])
	if strings.Index(commandKey, "!") != 0 {
		return
	}
	switch {
	// case u.IsAdmin && commandKey == "!kicks":
	// 	r.kicksHistoryCmd(u, b)
	// 	return
	case u.IsAdmin && commandKey == "!kara":
		r.punishUserCmd(u, b)
		return
	case commandKey == "!help":
		r.helpCmd(u, b)
		return
	case commandKey == "!strefy":
		msg := customeMsg.Strefy
		go b.exec(sendMessage("1", r.params[0]["invokerid"], msg))
		return
	case commandKey == "!accept":
		r.acceptCmd(u, b)
		return
	case commandKey == "!uptime":
		r.upTimeCmd(u, b)
		return
	case u.IsAdmin && commandKey == "!quit":
		r.quitCmd(u, b)
		return
	case u.IsAdmin && b.isMaster && commandKey == "!room":
		r.createChannelCmd(u, b)
		return
	case u.IsAdmin && b.isMaster && commandKey == "!create":
		infoLog.Println("Created by: ", b.ID)
		r.createNewBotCmd(u, b)
		return
	case u.IsAdmin && commandKey == "!check":
		r.checkIfRoomOutOfDateCmd(u, b)
		return
	case commandKey == "!addme":
		r.addUserAsAdminCmd(u, b)
		return
	case commandKey == "!settoken":
		r.setTokenCmd(u, b)
		return
	case commandKey == "!test":
		go b.exec(sendMessage("1", u.Clid, "Don't mind me I am just here for testing :)"))
		return
	case commandKey == "!turnoffpoke":
		r.togglePokeCmd(u, b)
		return
	case commandKey == "!turnofftext":
		r.togglePrivateMsgCmd(u, b)
		return
	case u.IsAdmin && commandKey == "!debuguser":
		r.debugUser(u, b)
		return
	case commandKey == "!token":
		r.recoverChannelAdminCmd(u, b)
		return
	default:
		warnLog.Println("User invoked unknow command - ", u.Nick, " commad was ", r.params[0]["msg"])
		go b.exec(sendMessage("1", u.Clid, "Jeśli widzisz tą wiadomość prawdopodbnie wpisałeś złą komende albo nie masz do niej dostępu."))
	}

}

//kicksHistoryAction, shows history of kicks on certain user
//!kicks uniqueID time.Time formated
//!kicks L6bv1FMnkDONcwnf3LMKEpcB5NU= 16-01-2017
// func (r *Response) kicksHistoryCmd(u *User, b *Bot) {
// 	kick := strings.SplitN(r.params[0]["msg"], " ", 3)
// 	if len(kick) != 3 {
// 		warnLog.Println("Akcja kick wymaga więcej parametrów", u.Nick)
// 		return
// 	}
// 	o, e := b.getUserKickBanHistory("kicks", kick[1], kick[2])
// 	if e != nil {
// 		errLog.Println(e)
// 	}
// 	go b.exec(sendMessage("1", r.params[0]["invokerid"], o))
// }

//punishUserCmd, moves user to sticky channel
//in which he needs to stay in for punish time
//in real time
func (r *Response) punishUserCmd(u *User, b *Bot) {
	punish := strings.SplitN(r.params[0]["msg"], " ", 3)
	if len(punish) != 3 {
		warnLog.Println("Akcja kara wymaga więcej parametrów", u.Nick)
		return
	}
	if cfg.PunishRoom == "" {
		errLog.Println("Pole punish room id jest puste!")
		return
	}
	_, err := strconv.Atoi(punish[1])
	if err != nil {
		errLog.Println("Pierwszy parametr nie był intem")
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Pierwszy parametr musi być liczbą"))
		return
	}

	res, e := b.exec(clientFind(punish[2]))
	if e != nil {
		errLog.Println("Nie ma takiego użytkownika w bazie danych", e)
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Podany użytkownik nie istnieje. Czy to na pewno[color=red][b] "+punish[2]+"[/b][/color]? Spróbuj ponownie."))
	}
	userClidb, userOK := usersByClid[res.params[0]["clid"]]
	if !userOK {
		return
	}
	user, k := users[userClidb]
	if !k || user.IsAdmin {
		return
	}
	user.Punish.IsPunished = true
	f, err := strconv.ParseFloat(punish[1], 64)
	if err != nil {
		errLog.Println(e)
	}
	if f == 0 {
		debugLog.Println("Anulowanie kary dla użytkownika", user.Nick)
		go b.exec(clientMove(u.Clid, cfg.GuestRoom))
		go b.exec(clientPoke(res.params[0]["clid"], "[color=red][b]Twoja kara została anulowana przez Admina[/b][/color]"))
		user.Punish.IsPunished = false
		return
	}
	user.Punish.OriginTime = f
	go PunishRoom(b, user)
	go b.exec(clientMove(res.params[0]["clid"], cfg.PunishRoom))
	go b.exec(clientPoke(res.params[0]["clid"], "[color=red][b]Otrzymałeś "+punish[1]+" sekund kary wczasie rzeczywistym na karnym jeżyku"))
	go b.exec(sendMessage("1", r.params[0]["invokerid"], "Użytkownik otrzymał karę na "+punish[1]+"sekund jeśli chcesz ją anulować wpisz !kara 0 "+punish[2]))
}

//helpCmd, shows info about available Commands
func (r *Response) helpCmd(u *User, b *Bot) {
	resGroup, e := b.exec(serverGroupIdsByCliDB(u.Clidb))
	if e != nil {
		errLog.Println("Error while group retriving in help command", e)
		return
	}
	msg := ""
	for _, v := range resGroup.params {
		if v["name"] == "Admin Server Query" || v["name"] == "Head Admin" {
			msg = customeMsg.CommandsAdmin
			break
		}
		msg = customeMsg.Commands
		break
	}
	go b.exec(sendMessage("1", r.params[0]["invokerid"], msg))
}

//acceptCmd, allows users to accept Terms of Service
func (r *Response) acceptCmd(u *User, b *Bot) {
	if u.BasicInfo.ReadRules {
		eventLog.Println(u.Nick, "już dokonał akceptacji regulaminu")
		return
	}
	u.BasicInfo.ReadRules = true
	b.db.AddRecord("users", u.Clidb, u)
	go b.exec(sendMessage("1", r.params[0]["invokerid"], customeMsg.RulesAccepted))
	go b.exec(serverGroupAddClient(cfg.TempGroup, u.Clidb))
	eventLog.Println(u.Nick, "zaakceptował regulamin")
}

//upTimeCmd, shows how much time bot is up
func (r *Response) upTimeCmd(u *User, b *Bot) {
	bot := strings.SplitN(r.params[0]["msg"], " ", 2)
	if len(bot) == 2 {
		botCheck, ok := bots[bot[1]]
		if ok && b.ID == bot[1] {
			t := time.Unix(botCheck.Uptime, 0)
			since := time.Since(t)
			if since < 3600 {
				since.Minutes()
			}
			eventLog.Println("Bot ", botCheck.ID, " uptime is", since.String())
			go b.exec(sendMessage("1", r.params[0]["invokerid"], "Jestem online od: "+since.String()))
			return
		}
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Nie ma takiego bota"))
	}
}

//quitCmd, shutdown bot
func (r *Response) quitCmd(u *User, b *Bot) {
	bot := strings.SplitN(r.params[0]["msg"], " ", 2)
	if len(bot) == 2 {
		botToClose, ok := bots[bot[1]]
		if ok {
			botToClose.conn.Close()
			warnLog.Println("User ", u.Nick, " invoked command to turn of bot")
			go b.exec(sendMessage("1", r.params[0]["invokerid"], "Bot zostaję wyłączony!"))
			return
		}
		errLog.Println("There is no such bot as ", bot[1])
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Podany bot nie istniej"))
	}
}

//createChannelCmd, creates channel for user
//!room <spacer> <user_name>
func (r *Response) createChannelCmd(u *User, b *Bot) {
	room := strings.SplitN(r.params[0]["msg"], " ", 3)
	if len(room) != 3 {
		return
	}
	pid, ok := isSpacer(room[1])
	if !ok {
		errLog.Println("No such spacer as ", room[1])
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Pokój nie został utworzony powód: podana strefa nie istnieje"))
		return
	}
	go func() {
		cid, errC := b.newRoom(room[2], pid, true, 0)
		if errC != nil {
			errLog.Println(errC)
			go b.exec(sendMessage("1", r.params[0]["invokerid"], "Pokój nie został utworzony powód: "+errC.Error()))
			return
		}
		if cid[0] != "" {
			cidChild, err := b.newRoom("", cid[0], false, 2)
			if err != nil {
				return
			}
			cid = append(cid, cidChild[0])
			cid = append(cid, cidChild[1])
		}
		cinfo, err := b.exec(clientFind(room[2]))
		if err != nil {
			errLog.Println("Client info command ", err)
			go b.exec(sendMessage("1", r.params[0]["invokerid"], "Pokój utworzony bez Channel Admina powód:  "+err.Error()))
		}
		dbID := getDBFromClid(cinfo.params[0]["clid"])
		if dbID == "" {
			errLog.Println("Client dbID is empty")
			go b.exec(sendMessage("1", r.params[0]["invokerid"], "Podczas tworzenia pokoju natrafiliśmy na bład, który wynika z braku użytkownika w mapie (zła nazwa użytkownika)"))
		}
		_, errS := b.exec(setChannelAdmin(dbID, cid[0]))
		if errS != nil {
			errLog.Println("Set Admin command: ", errC)
			go b.exec(sendMessage("1", r.params[0]["invokerid"], "Niepoprawna nazwa użytkownika zaskutkowała błędem przy nadawaniu praw Channel Admina użytkownikowi. Szczegóły: "+errS.Error()))
		}
		token := randString(7)
		v := &Token{Token: token, Cid: cid[0], LastChange: time.Now(), EditedBy: b.ID + " - room Created"}
		b.db.AddRecordSubBucket("rooms", "tokens", token, v)
		var admins []string
		admins = append(admins, dbID)
		channel := &Channel{
			Cid:        cid[0],
			Spacer:     pid,
			Name:       room[2],
			OwnerDB:    dbID,
			CreateDate: time.Now(),
			CreatedBy:  "",
			Token:      v.Token,
			Childs:     cid[1:],
			Admins:     admins,
		}
		b.db.AddRecord("rooms", cid[0], channel)
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Pokój o nazwie "+channel.Name+" z tokenem  [b][color=red]"+v.Token+" [/color][/b]został sukcesywnie utworzony!"))
		if cinfo.params[0]["clid"] != "" {
			go b.exec(sendMessage("1", cinfo.params[0]["clid"], "Token dla Twojego kanału to [b][color=red]"+v.Token+" [/color][/b]służy on do odzyskania channel Admina."))
		} else {
			go b.exec(sendMessage("1", r.params[0]["clid"], "Nazwa pokoju utworzonego przez Ciebie nie zawiera poprawnej nazwy użytkownika, proszę wysłać mu token, który otrzymałeś w wiadomości prywatnej by naprawić ten błąd"))
		}
	}()
}

//createNewBotCmd, creates an extra bot
func (r *Response) createNewBotCmd(u *User, b *Bot) {
	bot := &Bot{}
	err := bot.newBot("teamspot.eu:10011", false)
	if err != nil {
		go b.exec(sendMessage("1", u.Clid, err.Error()))
		log.Println(err)
		return
	}

	eventLog.Println("New bot with id: ", bot.ID, "total of", len(bots), "in system", "bot created by ", u.Nick)
	bot.execAndIgnore(cmdsSub, true)
	go b.exec(sendMessage("1", u.Clid, "Nowy bot o nazwie "+bot.ID+" został utworzony bez problemów"))
}

//checkIfRoomOutOfDateCmd, manually invoke command
//which checks if rooms are not used more then 14days
func (r *Response) checkIfRoomOutOfDateCmd(u *User, b *Bot) {
	clean := strings.SplitN(r.params[0]["msg"], " ", 2)
	if len(clean) != 2 {
		return
	}
	if clean[1] == "true" {
		go b.checkIfRoomOutDate(true, "0")
		return
	}
	b.checkIfRoomOutDate(false, r.params[0]["invokerid"])
	return
}

//addUserAsAdminCmd, adds user to admin list if he has admin group
func (r *Response) addUserAsAdminCmd(u *User, b *Bot) {
	userGroups, e := b.exec(serverGroupIdsByCliDB(u.Clidb))
	if e != nil {
		errLog.Println(e)
		return

	}
	for _, uGroup := range userGroups.params {
		if uGroup["name"] == "Head Admin" || uGroup["name"] == "Admin Server Query" {
			if u.IsAdmin {
				eventLog.Println(u.Nick, "jest już wpisany jako administrator!")
				return
			}
			u.IsAdmin = true
			b.db.AddRecord("users", u.Clidb, u)
			eventLog.Println("Dodano użytkownika", u.Nick, "do listy adminów na bocie.")
			go b.exec(sendMessage("1", u.Clid, "Zostałeś dodany do listy administracji"))
			return
		}
	}
	warnLog.Println("Nieautoryzowana próba dodania do listy adminów przez użytkownika", u.Nick)
	go b.exec(sendMessage("1", u.Clid, "Niestety nie spełniasz wymagań :("))
}

//setTokenCmd, allows old users and users who doesn't have token
//on thier channel to set it, they need to have channel admin on channel
func (r *Response) setTokenCmd(u *User, b *Bot) {
	chName := strings.SplitN(r.params[0]["msg"], " ", 2)
	if len(chName) != 2 {
		return
	}

	cInfo, err := b.exec(channelFind(chName[1]))
	if err != nil {
		errLog.Println(err)
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Podałeś niepoprawną nazwę pokoju"))
		return
	}
	debugLog.Println(cInfo.params[0]["cid"])
	cOwner, er := b.exec(getChannelAdmin(cInfo.params[0]["cid"]))
	if er != nil {
		eventLog.Println(er)
		return
	}

	for _, o := range cOwner.params {
		if o["cldbid"] != u.Clidb {
			continue
		}

		chInfo, e := b.exec(channelInfo(cInfo.params[0]["cid"]))
		if e != nil {
			errLog.Println("error while channel retriving", e)
			return
		}

		for _, s := range cfg.Spacer {
			if chInfo.params[0]["pid"] != s {
				continue
			}

			room, e := b.db.GetRecord("rooms", cInfo.params[0]["cid"])
			if e != nil {
				return
			}

			if len(room) != 0 {
				channel := &Channel{}
				channel.unmarshalJSON(room)

				if channel.Token == "" {
					token := randString(7)
					v := &Token{Token: token, Cid: cInfo.params[0]["cid"], LastChange: time.Now(), EditedBy: b.ID + " - room Created"}
					channel.Token = token
					b.db.AddRecord("rooms", channel.Cid, channel)
					b.db.AddRecordSubBucket("rooms", "tokens", token, v)
					go b.exec(sendMessage("1", r.params[0]["invokerid"], "Twój token został pomyślnie utworzony [color=red][b]"+token+"[/b][/color] zapisz go sobie gdzieś"))

					return
				}
				eventLog.Println("Token dla pokoju", channel.Name, "już istnieje:", channel.Token)
				go b.exec(sendMessage("1", r.params[0]["invokerid"], "Token dla tego pokoju już istnieje [color=red][b]"+channel.Token+"[/b][/color]"))
			}
			return
		}
	}
}

//debugUser, shows current raw memory info about certain user
func (r *Response) debugUser(u *User, b *Bot) {
	dUser := strings.SplitN(r.params[0]["msg"], " ", 2)
	if len(dUser) != 2 {
		return
	}
	var userBuffer bytes.Buffer
	for _, v := range users {
		if v.Nick == dUser[1] {
			out, e := json.Marshal(v)
			if e != nil {
				errLog.Println(e)
			}
			userBuffer.WriteString(string(out))
		}
	}
	if userBuffer.String() == "" {
		go b.exec(sendMessage("1", r.params[0]["invokerid"], "Nie ma takiego użytkownika"))
		return
	}
	go b.exec(sendMessage("1", r.params[0]["invokerid"], "\n"+userBuffer.String()))
}

//recoverChannelAdminCmd, allows users to recover channel admin
//via token which was created whit thier channel or by !setToken
func (r *Response) recoverChannelAdminCmd(u *User, b *Bot) {
	token := strings.SplitN(r.params[0]["msg"], " ", 2)
	if len(token) == 2 {
		t, e := b.db.GetRecordSubBucket("rooms", "tokens", token[1])
		if e != nil {
			errLog.Println("Database error: ", e)
			return
		}
		sToken, ok := u.checkTokeAttempts(t)
		if !ok {
			debugLog.Println(sToken)
			go b.exec(sendMessage("1", r.params[0]["invokerid"], sToken))
			return
		}
		tok := &Token{}
		e = tok.unmarshalJSON(t)
		if e != nil {
			errLog.Println("unmarshal error: ", e)
			return
		}
		go b.exec(sendMessage("1", r.params[0]["invokerid"], sToken))
		go b.exec(setChannelAdmin(u.Clidb, tok.Cid))
		infoLog.Println("User", u.Nick, " requested channel admin assign with valid token", tok.Token)

	}
}

func (r *Response) togglePokeCmd(u *User, b *Bot) {
	poke := strings.SplitN(r.params[0]["msg"], " ", 2)
	if len(poke) != 2 {
		return
	}
	id, e := b.exec(getPermissionID("i_client_needed_poke_power"))
	if e != nil {
		errLog.Println("Wystąpił błąd przy pobieraniu id pozwolenia")
		return
	}
	value := "50"
	if poke[1] == "on" {
		value = "85"
	}
	_, er := b.exec(addPermission(u.Clidb, id.params[0]["permid"], value, "0"))
	if er != nil {
		errLog.Println(e)
	}
}

//i_client_needed_private_textmessage_power
func (r *Response) togglePrivateMsgCmd(u *User, b *Bot) {
	text := strings.SplitN(r.params[0]["msg"], " ", 2)
	if len(text) != 2 {
		return
	}
	id, e := b.exec(getPermissionID("i_client_needed_private_textmessage_power"))
	if e != nil {
		errLog.Println("Wystąpił błąd przy pobieraniu id pozwolenia")
		return
	}
	value := "50"
	if text[1] == "on" {
		value = "85"
	}
	_, er := b.exec(addPermission(u.Clidb, id.params[0]["permid"], value, "0"))
	if er != nil {
		errLog.Println(e)
	}
}
