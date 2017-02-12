package main

import (
	"strconv"
	"strings"
	"time"
)

func (b *Bot) notifyAction(r *Response) {
	switch r.action {
	case "notifytextmessage":
		cinfo, ok := usersByClid[r.params[0]["invokerid"]]
		if !ok {
			return
		}
		user, ok := users[cinfo]
		if !ok {
			return
		}
		if strings.Index(r.params[0]["msg"], "!") == 0 {
			b.actionMsg(r, user)
			return
		}
		eventLog.Println("Text message: ", r.params[0]["msg"])
	case "notifyclientmoved":
		//This if statment prevents from adding fakes moves to jump protection counter by admin move option
		if r.params[0]["invokername"] != "" || r.params[0]["invokerid"] != "" {
			eventLog.Println("Admin przeniósł użytkownika o id", r.params[0]["clid"], "na kanał", r.params[0]["ctid"])
			return
		}
		b.jumpProtection(r)
	case "notifychanneledited":
		//Add edition note in channel description
		//For more info who edited the channel
	case "notifycliententerview":
		r.onClientEnterView(b)
	case "notifyclientleftview":
		if r.params[0]["reasonmsg"] == "" {
			go b.exec(kickClient(r.params[0]["invokerid"], "Wypełniaj powód kick/ban"))
		}
		if r.params[0]["invokeruid"] == "serveradmin" {
			return
		}
		if r.params[0]["reasonmsg"] == "deselected virtualserver" {
			infoLog.Println("Sever query bot event deselected virtualserver")
			return
		}
		userClid, ok := usersByClid[r.params[0]["clid"]]
		if !ok {
			if r.params[0]["reasonmsg"] == "connection lost" {
				debugLog.Println("Probably bot turning off", r.params)
				return
			}
			warnLog.Println("Abnormal action")
			return
		}
		user, ok := users[userClid]
		if !ok {
			warnLog.Println("Abnormal action")
			return
		}

		if r.params[0]["reasonid"] == "5" {
			infoLog.Println(user.Nick, "kicked from server by", r.params[0]["invokername"])
			// user.BasicInfo.Kick++
			// b.addKickBan("kicks", user.Clidb, r.params[0]["reasonmsg"], r.params[0]["invokername"])

		}

		if r.params[0]["reasonid"] == "6" {
			infoLog.Println(user.Nick, "banned from server by", r.params[0]["invokername"])
			// user.BasicInfo.Ban++
			// b.addKickBan("bans", user.Clidb, r.params[0]["reasonmsg"], r.params[0]["invokername"])
		}

		if r.params[0]["reasonid"] == "4" {
			infoLog.Println(user.Nick, " kicked from channel", r.params[0]["cid"])
		}

		b.db.AddRecord("users", user.Clidb, user)
		delete(users, user.Clidb)
		delete(usersByClid, r.params[0]["clid"])
	case "notifychanneldescriptionchanged":
		return
	case "notifychannelcreated":
		if r.params[0]["invokeruid"] != "serveradmin" {
			eventLog.Println("Room created manually by user:", r.params[0]["invokername"], "room name: ", r.params[0]["channel_name"])
			go b.roomFromNotify(r)
		} else {
			eventLog.Println("Room: ", r.params[0]["channel_name"], "created by bot", b.ID)
		}
		return
	case "notifychannelmoved":
		eventLog.Println("User", r.params[0]["invokername"], "moved channel ", r.params[0]["cid"], "to ", r.params[0]["cpid"])
		room, e := b.db.GetRecord("rooms", r.params[0]["cid"])
		if e != nil {
			errLog.Println("Database error:", e)
		}
		if len(room) != 0 {
			channel := &Channel{}
			channel.unmarshalJSON(room)
			channel.Spacer = r.params[0]["cpid"]
			err := b.db.AddRecord("rooms", channel.Cid, channel)
			if err != nil {
				errLog.Println("Database error:", e)
			}

		}
		return
	case "notifychanneldeleted":
		warnLog.Println("Room ", r.params[0]["cid"], "deleted by", r.params[0]["invokername"])
		delChannel := &DelChannel{
			Cid:        r.params[0]["cid"],
			DeletedBy:  r.params[0]["invokername"],
			InvokerUID: r.params[0]["invokeruid"],
			DeleteDate: time.Now(),
		}
		b.db.AddRecord("deletedRooms", delChannel.Cid, delChannel)
		_, err := b.db.GetRecord("rooms", r.params[0]["cid"])
		if err != nil {
			errLog.Println("No such room in databse cannot delete it from it")
			return
		}
		err = b.db.DeleteRecord("rooms", delChannel.Cid)
		if err != nil {
			errLog.Println(err)
		}
	default:
		warnLog.Println("Unusual action: ", r.action)
		return
	}

}

func (r *Response) onClientEnterView(b *Bot) {
	userDB, err := b.db.GetRecord("users", r.params[0]["client_database_id"])
	if err != nil {
		errLog.Println(err)
	}

	if len(userDB) != 0 {
		retriveUser := &User{}
		retriveUser.unmarshalJSON(userDB)
		retriveUser.Clid = r.params[0]["clid"]
		retriveUser.Nick = r.params[0]["client_nickname"]
		users[retriveUser.Clidb] = retriveUser
		usersByClid[r.params[0]["clid"]] = retriveUser.Clidb
		if time.Since(retriveUser.Moves.SinceMove).Seconds() > 600 {
			retriveUser.Moves.Number = 0
		}

		if time.Since(retriveUser.BasicInfo.CreatedAT).Seconds() > 172800 {
			retriveUser.BasicInfo.IsRegistered = true
		}

		if retriveUser.BasicInfo.ReadRules == false {
			go func() {
				msg := customeMsg.RuleOne
				b.exec(sendMessage("1", retriveUser.Clid, msg))
				time.Sleep(2 * time.Nanosecond)
				msg = customeMsg.RuleTwo
				b.exec(sendMessage("1", retriveUser.Clid, msg))
			}()
		}
		if retriveUser.Punish.IsPunished == true {
			go b.exec(clientMove(retriveUser.Clid, cfg.PunishRoom))
			go PunishRoom(b, retriveUser)
			retriveUser.Punish.OriginTime = (retriveUser.Punish.OriginTime - retriveUser.Punish.CurrentTime) + (10 * float64(retriveUser.Moves.Warnings))
			timeLeft := retriveUser.Punish.OriginTime - retriveUser.Punish.CurrentTime
			strLeft := strconv.FormatFloat(timeLeft, 'f', 1, 64)
			go b.exec(clientMove(retriveUser.Clid, cfg.PunishRoom))
			go b.exec(clientPoke(retriveUser.Clid, "[color=red][b]Zostało Ci jeszcze "+strLeft+" sekund na karnym jeżyku, powodzenia :)[b][/color]"))
			eventLog.Println(retriveUser.Nick, "nie odbył pełnej kary na karnym jeżyku, zostało mu jeszcze", strLeft, "sekund")
		}

	} else {
		if r.params[0]["client_database_id"] != "1" && r.params[0]["client_unique_identifier"] != "ServerQuery" {
			userS := createUser(r.params[0]["client_database_id"], r.params[0]["clid"], r.params[0]["client_nickname"])
			b.db.AddRecord("users", r.params[0]["client_database_id"], userS)
			users[userS.Clidb] = userS
			usersByClid[userS.Clid] = userS.Clidb
			go func() {
				msg := customeMsg.RuleOne
				b.exec(sendMessage("1", r.params[0]["clid"], msg))
				time.Sleep(2 * time.Nanosecond)
				msg = customeMsg.RuleTwo
				b.exec(sendMessage("1", r.params[0]["clid"], msg))
			}()

		} else {
			warnLog.Println("Detected sever query bot")
		}
	}
}
