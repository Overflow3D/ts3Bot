package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"

	_db "github.com/overflow3d/ts3_/database"
)

//Config , TeamSpeak 3 bot start up
type Config struct {
	Login          string            `json:"Login"`
	Password       string            `json:"Password"`
	ServerID       string            `json:"ServerID"`
	HeadAdmin      string            `json:"HeadAdminCliDB"`
	Spacer         map[string]string `json:"Spacer"`
	ChannelAdmin   string            `json:"ChannelAdmin"`
	BotMainChannel string            `json:"BotMainChannel"`
	PunishRoom     string            `json:"PunishRoom"`
	GuestRoom      string            `json:"GuestRoom"`
	TempGroup      string            `json:"TempGroup"`
	PermGroup      string            `json:"PermGroup"`
}

//Messages , Load all custome messages
type Messages struct {
	RuleOne       string `json:"RuleOne"`
	RuleTwo       string `json:"RuleTwo"`
	Commands      string `json:"Commands"`
	CommandsAdmin string `json:"CommandsAdmin"`
	Strefy        string `json:"Strefy"`
}

var (
	wg         sync.WaitGroup
	cmdsMain   []*Command
	cmdsSub    []*Command
	cfg        *Config
	customeMsg *Messages
)

func main() {
	// After cloning the git you need to create config.json file
	// If you want to use loadConfig function, otherwise you will get an error

	var err error
	cfg, err = loadConfig()
	if err != nil {
		errLog.Println(err)
	}

	if len(cfg.Spacer) == 0 {
		errLog.Println("Musisz skonfigurować spacery przed odpaleniem bota!")
		return
	}
	//Basic commands to start off bot
	cmdsMain = []*Command{
		useServer(cfg.ServerID),
		logIn(cfg.Login, cfg.Password),
		nickname("SkyNet"),
		notifyRegister("channel", "0"),
		notifyRegister("textchannel", "0"),
		notifyRegister("textprivate", "0"),
	}

	cmdsSub = []*Command{
		useServer(cfg.ServerID),
		logIn(cfg.Login, cfg.Password),
		notifyRegister("textchannel", ""),
		notifyRegister("textprivate", ""),
	}
	loadMessages()
	b := &Bot{}
	db, err := _db.NewConn()
	defer db.Close()
	if err != nil {
		errLog.Println(err)
	}
	b.db = db
	err = b.newBot("teamspot.eu:10011", true)
	if err != nil {
		errLog.Println(err)
	}
	bot, ok := bots["master"]
	if ok {
		bot.execAndIgnore(cmdsMain, false)
		err := bot.loadUsers()
		bot.getChannelList()
		if err != nil {
			log.Fatalln(err)
		}
	}

	wg.Wait()

}

//Loading config from json file
func loadConfig() (*Config, error) {
	cfg := &Config{}
	config, err := ioutil.ReadFile("./config.json")
	if err != nil {
		return nil, err
	}
	err = cfg.unMarshalJSON(config)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadMessages() {
	msg := &Messages{}
	m, err := ioutil.ReadFile("./message.json")
	if err != nil {
		errLog.Println("Message loading", err)
		return
	}
	err = msg.unMarshalJSON(m)
	if err != nil {
		errLog.Println("Message loading", err)
		return
	}
	customeMsg = msg
	infoLog.Println("Custome message loaded")
}

//Unmarshaling json into Cofig struct
func (c *Config) unMarshalJSON(data []byte) error {
	return json.Unmarshal(data, &c)
}

func (c *Config) marshalJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

func (m *Messages) unMarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m)
}
