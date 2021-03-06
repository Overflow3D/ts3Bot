package main

import (
	"bytes"
	"fmt"
	"log"
)

//Command , struct for custome commands
type Command struct {
	Name   string
	params map[string]string
	flags  []string
}

//Converts struct to string
func (c *Command) String() (cmd string) {
	var b bytes.Buffer
	b.WriteString(c.Name + " ")
	for key, v := range c.params {
		b.WriteString(key + "=")
		b.WriteString(escape(v) + " ")
	}

	for _, v := range c.flags {
		b.WriteString(escape(v) + " ")
	}

	return b.String()
}

//Exec single command
func (b *Bot) exec(cmd *Command) (*Response, error) {

	fmt.Fprintf(b.conn, "%s\n\r", cmd)
	err := <-b.err
	res := b.resp
	b.resp = ""
	return formatResponse(res, "cmd"), formatError(err)
}

//Exec multiple commands and ignore output silently
func (b *Bot) execAndIgnore(cmd []*Command, newBot bool) {
	for _, c := range cmd {
		fmt.Fprintf(b.conn, "%s\n\r", c)
		err := <-b.err
		b.resp = ""
		if formatError(err) != nil {
			log.Println(err)
			return
		}
	}

	if newBot {
		b.exec(nickname(b.ID))
	}
	r, e := b.exec(whoami())
	if e != nil {
		return
	}
	b.exec(clientMove(r.params[0]["client_id"], cfg.BotMainChannel))

	infoLog.Println("Command executed with out any problems, invoked by bot: ", b.ID)

}

func version() *Command {
	return &Command{
		Name: "version",
	}
}

//DEFAULT TEAMSPEAK3 COMMANDS

func whoami() *Command {
	return &Command{
		Name: "whoami",
	}
}

func useServer(id string) *Command {
	return &Command{
		Name: "use 4",
		params: map[string]string{
			"sid": id,
		},
	}
}

func logIn(login string, pass string) *Command {
	return &Command{
		Name: "login",
		params: map[string]string{
			"client_login_name":     login,
			"client_login_password": pass,
		},
	}
}

func channelList() *Command {
	return &Command{
		Name: "channellist",
	}
}

func clientMove(clid, cid string) *Command {
	return &Command{
		Name: "clientmove",
		params: map[string]string{
			"clid": clid,
			"cid":  cid,
		},
	}
}

func notifyRegister(e string, id string) *Command {
	if id != "" {
		return &Command{
			Name: "servernotifyregister",
			params: map[string]string{
				"event": e,
				"id":    id, //register to channel 0, for more events
			},
		}
	}
	return &Command{
		Name: "servernotifyregister",
		params: map[string]string{
			"event": e,
		},
	}
}

func kickClient(clid string, reason string) *Command {
	return &Command{
		Name: "clientkick",
		params: map[string]string{
			"clid":      clid,
			"reasonid":  "5",
			"reasonmsg": reason,
		},
	}
}

func clientDBID(uid string, flag string) *Command {
	return &Command{
		Name: "clientdbfind",
		params: map[string]string{
			"pattern": uid,
		},
		flags: []string{flag},
	}
}

func createRoom(name string, pid string) *Command {
	return &Command{
		Name: "channelcreate",
		params: map[string]string{
			"channel_name":           name,
			"channel_flag_permanent": "1",
			"cpid": pid,
		},
	}
}

func nickname(user string) *Command {
	return &Command{
		Name: "clientupdate",
		params: map[string]string{
			"client_nickname": user,
		},
	}
}

func clientInfo(clid string) *Command {
	return &Command{
		Name: "clientinfo",
		params: map[string]string{
			"clid": clid,
		},
	}
}

func clientList() *Command {
	return &Command{
		Name: "clientlist",
	}
}

func getChannelAdmin(cid string) *Command {
	return &Command{
		Name: "channelgroupclientlist",
		params: map[string]string{
			"cid":  cid,
			"cgid": "18",
		},
	}
}

func setChannelAdmin(cldbid string, cid string) *Command {
	return &Command{
		Name: "setclientchannelgroup",
		params: map[string]string{
			"cgid":   cfg.ChannelAdmin,
			"cid":    cid,
			"cldbid": cldbid,
		},
	}
}

//targetMode 1-client, 2-channel, 3-server
//target cid
func sendMessage(targetMode string, target string, msg string) *Command {
	return &Command{
		Name: "sendtextmessage",
		params: map[string]string{
			"targetmode": targetMode,
			"target":     target,
			"msg":        msg,
		},
	}
}

func sendOffline(cluid, subject, msg string) *Command {
	return &Command{
		Name: "messageadd",
		params: map[string]string{
			"cluid":   cluid,
			"subject": subject,
			"message": msg,
		},
	}
}

func clientFind(user string) *Command {
	return &Command{
		Name: "clientfind",
		params: map[string]string{
			"pattern": user,
		},
	}
}

func channelInfo(cid string) *Command {
	return &Command{
		Name: "channelinfo",
		params: map[string]string{
			"cid": cid,
		},
	}
}

func deleteChannel(cid string) *Command {
	return &Command{
		Name: "channeldelete",
		params: map[string]string{
			"cid":   cid,
			"force": "1",
		},
	}
}

func serverGroupIdsByCliDB(id string) *Command {
	return &Command{
		Name: "servergroupsbyclientid",
		params: map[string]string{
			"cldbid": id,
		},
	}
}

func channelFind(name string) *Command {
	return &Command{
		Name: "channelfind",
		params: map[string]string{
			"pattern": name,
		},
	}
}

func serverGroupAddClient(groupID string, cldbid string) *Command {
	return &Command{
		Name: "servergroupaddclient",
		params: map[string]string{
			"sgid":   groupID,
			"cldbid": cldbid,
		},
	}
}

func serverGroupDelClient(groupID string, cldbid string) *Command {
	return &Command{
		Name: "servergroupdelclient",
		params: map[string]string{
			"sgid":   groupID,
			"cldbid": cldbid,
		},
	}
}

func clientPoke(id, msg string) *Command {
	return &Command{
		Name: "clientpoke",
		params: map[string]string{
			"clid": id,
			"msg":  msg,
		},
	}
}

func getPermissionID(name string) *Command {
	return &Command{
		Name: "permidgetbyname",
		params: map[string]string{
			"permsid": name,
		},
	}
}

func addPermission(id, permID, value, skip string) *Command {
	return &Command{
		Name: "clientaddperm",
		params: map[string]string{
			"cldbid":    id,
			"permid":    permID,
			"permvalue": value,
			"permskip":  skip,
		},
	}
}
