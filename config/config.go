package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type GlobalConfig struct {
	Uuid              float64           `json:"uuid"`
	ConnectionsConfig ConnectionsConfig `json:"ConnectionsConfig"`
}

type ConnectionsConfig struct {
	KafkaServer    KafkaConsumer  `json:"KafkaConsumer"`
	Client         Client         `json:"Client"`
	Mongo          Mongo          `json:"Mongo"`
	EventProcessor EventProcessor `json:"EventProcessor"`
}

type KafkaConsumer struct {
	ServerUrls            []string `json:"server_urls"`
	CGroup                string   `json:"c_group"`
	ConnectionPullSize    int      `json:"connection_pull_size"`
	Topics                []string `json:"topics"`
}

type EventProcessor struct {
	Url string `json:"url"`
}

type Client struct {
	ListenPort            string `json:"listen_port"`
	MaxConnectionPoolSize int    `json:"max_connection_pool_size"`
	MaxBuffSize           int    `json:"max_buff_size"`
}

type Mongo struct {
	Uri         string   `json:"uri"`
	Database    string   `json:"database"`
}

func GetConfig() GlobalConfig {
	return *globalConfig
}

var globalConfig *GlobalConfig

func init() {

	log.Println("Creating configuration...")

	log.Println("Reading config.json file...")
	fileRaw, err := ioutil.ReadFile("./resources/config.json")

	if err != nil {
		log.Fatal("Can not read from config.json file", err)
	} else {
		log.Println("Reading successful")
	}

	log.Println("Validating config.json file...")
	isValid := json.Valid(fileRaw)
	if !isValid {
		log.Fatal("Invalid config.json file format. JSON structure expected expected")
	} else {
		log.Println("Validating successful")
	}

	log.Println("Decoding config.json file... ")
	err = json.Unmarshal(fileRaw, &globalConfig)
	if err != nil {
		log.Fatal("Decoding failed ", err)
	} else {
		log.Println("Decoding successful")
	}

}
