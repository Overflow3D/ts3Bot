package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

//Config, TeamSpeak 3 bot start up
type Config struct {
	Login    string `jsong:"Login"`
	Password string `json:"Password"`
	ServerID int    `json:"ServerID"`
}

func main() {
	config, err := loadConfig()
	if err != nil {
		panic(err)
	}
	log.Println(config)
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
