package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
)

//Config , TeamSpeak 3 bot start up
type Config struct {
	Login    string `jsong:"Login"`
	Password string `json:"Password"`
	ServerID string `json:"ServerID"`
}

var wg sync.WaitGroup
var cmdsMain []*Command
var cmdsSub []*Command

func main() {
	// After cloning the git you need to create config.json file
	// If you want to use loadConfig function, otherwise you will get an error

	cfg, err := loadConfig()
	if err != nil {
		log.Println(err)
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
		notifyRegister("channel", ""),
		notifyRegister("textchannel", ""),
		notifyRegister("textprivate", ""),
	}

	newBot("teamspot.eu:10011", true)
	bot, ok := bots["master"]
	if ok {
		bot.execAndIgnore(cmdsMain)
		bot.loadUsers()
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

//Unmarshaling json into Cofig struct
func (c *Config) unMarshalJSON(data []byte) error {
	return json.Unmarshal(data, &c)
}
