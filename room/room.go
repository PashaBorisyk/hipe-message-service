package room

import (
	"partyfy-message-service/config"
	"sync"
)

const (
	UserID = "user_id"
)

type Room struct {
	clientCounter        int
	serverError          error
	thisPort             string
	kafkaAddress         []string
	clientBuffSize       int
	serverBuffSize       int
	maxClientConnections int
	channelsCount        int
	userChannels         map[int]*MessageChannel         //userID -> MessageChannel
	waitGroup            *sync.WaitGroup
	config               config.GlobalConfig
}

func NewRoomFromConfig(waitGroup *sync.WaitGroup) *Room {

	var globalConfig = config.GetConfig()

	return &Room{
		thisPort:             globalConfig.ConnectionsConfig.Client.ListenPort,
		kafkaAddress:         globalConfig.ConnectionsConfig.KafkaServer.ServerUrls,
		clientBuffSize:       globalConfig.ConnectionsConfig.Client.MaxBuffSize,
		userChannels:         make(map[int]*MessageChannel, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		waitGroup:            waitGroup,
		maxClientConnections: globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize,
		config:               globalConfig,
	}

}
