package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

var globalConfig *GlobalConfig

func init() {

	log.Println("Creating configuration...")

	log.Println("Reading config.json file...")
	fileRaw, err := ioutil.ReadFile("./resources/config.json")

	if err != nil {
		log.Println("Can not read from config.json file")
		log.Print(err)
	} else {
		log.Println("Reading successful")
	}

	log.Println("Validating config.json file...")
	isValid := json.Valid(fileRaw)
	if !isValid {
		log.Print("Invalid config.json file format. JSON structure expected expected")
	} else {
		log.Println("Validating successful")
	}

	log.Println("Decoding config.json file...")
	var config GlobalConfig
	err = json.Unmarshal(fileRaw, &config)
	if err != nil {
		log.Println("Decoding failed")
	} else {
		log.Println("Decoding successfull")
		log.Println(config)
		globalConfig = &config
	}

}

type GlobalConfig struct {
	Uuid              float64           `json:"uuid"`
	ConnectionsConfig ConnectionsConfig `json:"ConnectionsConfig"`
}

type ConnectionsConfig struct {
	KafkaServer    KafkaConsumer  `json:"KafkaConsumer"`
	Client         Client         `json:"Client"`
	Room           Room           `json:"Room"`
	Mongo          Mongo          `json:"Mongo"`
	EventProcessor EventProcessor `json:"EventProcessor"`
}

type KafkaConsumer struct {
	ServerUrls            []string `json:"server_urls"`
	CGroup                string   `json:"c_group"`
	ConnectionPullSize    int      `json:"connection_pull_size"`
	MaxConnectionPoolSize int      `json:"max_connection_pool_size"`
	MinConnectionPoolSize int      `json:"min_connection_pool_size"`
	AutoConfigPoolSize    bool     `json:"auto_config_pool_size"`
	Topics                []string `json:"topics"`
}

type EventProcessor struct {
	Url string `json:"url"`
}

type Client struct {
	ListenPort            string `json:"listen_port"`
	ConnectionPullSize    int    `json:"connection_pull_size"`
	MaxConnectionPoolSize int    `json:"max_connection_pool_size"`
	MinConnectionPoolSize int    `json:"min_connection_pool_size"`
	MaxBuffSize           int    `json:"max_buff_size"`
	AutoConfigPoolSize    bool   `json:"auto_config_pool_size"`
}

type Room struct {
	EventsMaxSize int `json:"events_max_size"`
	Category      int `json:"category"`
}

type Topics struct {
	TopicName string `json:"name"`
}

type Mongo struct {
	Uri         string   `json:"uri"`
	Database    string   `json:"database"`
	Collections []string `json:"collections"`
}

func GetConfig() GlobalConfig {
	return *globalConfig
}
