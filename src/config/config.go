package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

var globalConfig *GlobalConfig

func init() {

	log.Println("Creating configuration...")

	log.Println("Reading config.properties file...")
	fileRaw, err := ioutil.ReadFile("./resources/config.properties")

	if err != nil {
		log.Println("Can not read from config.properties file")
		log.Print(err)
	} else {
		log.Println("Reading successful")
	}

	log.Println("Validating config.properties file...")
	isValid := json.Valid(fileRaw)
	if !isValid {
		log.Print("Invalid config.properties file format. JSON structure expected expected")
	} else {
		log.Println("Validating successful")
	}

	log.Println("Decoding config.properties file...")
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
	Server Server `json:"Server"`
	Client Client `json:"Client"`
	Room   Room   `json:"Room"`
}

type Server struct {
	Url                   string `json:"url"`
	ConnectionPullSize    int    `json:"connection_pull_size"`
	MaxConnectionPoolSize int    `json:"max_connection_pool_size"`
	MinConnectionPoolSize int    `json:"min_connection_pool_size"`
	AutoConfigPoolSize    bool   `json:"auto_config_pool_size"`
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
}

func GetConfig() GlobalConfig {
	return *globalConfig
}
