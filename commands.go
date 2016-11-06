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

func (c *Command) String() (cmd string) {
	var b bytes.Buffer
	b.WriteString(c.Name + " ")
	for key, v := range c.params {
		b.WriteString(key + "=")
		b.WriteString(v + " ")
	}

	for _, v := range c.flags {
		b.WriteString(v + " ")
	}

	return b.String()
}

func (b *Bot) exec(cmd Command) (*Response, error) {
	fmt.Fprintf(b.conn, "%s\n\r", cmd)
	err := <-b.err
	res := b.resp
	b.resp = ""
	return formatResponse(res, "cmd"), formatError(err)
}

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

func useServer(id string) *Command {
	return &Command{
		Name: "use 4",
		params: map[string]string{
			"sid": id,
		},
	}
}

func logIn(l string, p string) *Command {
	return &Command{
		Name: "login",
		params: map[string]string{
			"client_login_name":     l,
			"client_login_password": p,
		},
	}
}
