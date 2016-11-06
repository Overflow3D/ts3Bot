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

func main() {
	// After cloning the git you need to create config.json file
	// If you want to use loadConfig function, otherwise you will get an error
	cfg, err := loadConfig()
	if err != nil {
		log.Println(err)
	}
	cmds := []*Command{
		useServer(cfg.ServerID),
		logIn(cfg.Login, cfg.Password),
	}

	newBot("teamspot.eu:10011", true)
	bot, ok := bots["master"]
	if ok {
		bot.execAndIgnore(cmds)
		// bot.writeToCon("channellist")
		// bot.writeToCon("servernotifyregister event=server")
	}

	wg.Wait()

}

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

func (c *Config) unMarshalJSON(data []byte) error {
	return json.Unmarshal(data, &c)
}
