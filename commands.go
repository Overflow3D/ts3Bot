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
func (b *Bot) execAndIgnore(cmd []*Command) {
	for _, c := range cmd {
		fmt.Fprintf(b.conn, "%s\n\r", c)
		err := <-b.err
		b.resp = ""
		if formatError(err) != nil {
			log.Println(err)
			return
		}
	}

	log.Println("Command executed with out any problems, invoked by bot: ", b.ID)

}

//Useless version as dummy ping
//Not useless if you want to upgrade ts automaticly
//But then you can change the code =)
func version() *Command {
	return &Command{
		Name: "version",
	}
}

//DEFAULT TEAMSPEAK3 COMMANDS

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

func clientDBID(uid string) *Command {
	return &Command{
		Name: "clientdbfind",
		params: map[string]string{
			"pattern": uid,
		},
		flags: []string{"-uid"},
	}
}
