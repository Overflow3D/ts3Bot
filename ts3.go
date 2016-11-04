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
	// config, err := loadConfig()
	// if err != nil {
	// 	panic(err)
	// }
	newBot("teamspot.eu:10011")
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
