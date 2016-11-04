package main

import (
	"encoding/json"
	"io/ioutil"
	"sync"
)

//Config , TeamSpeak 3 bot start up
type Config struct {
	Login    string `jsong:"Login"`
	Password string `json:"Password"`
	ServerID int    `json:"ServerID"`
}

var wg sync.WaitGroup

func main() {

	newBot("teamspot.eu:10011", true)
	bot, ok := bots["master"]
	if ok {

		bot.writeToCon("use 4")
		bot.writeToCon("version")
		bot.writeToCon("channellist")

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
