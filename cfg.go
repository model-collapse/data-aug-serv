package main

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	ObjectDir     string `json:"object_dir"`
	BackgroundDir string `json:"back_dir"`
}

var GConf Config

func LoadConfig(path string) (err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &GConf)
	return
}

