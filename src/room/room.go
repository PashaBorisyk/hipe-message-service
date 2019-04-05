package room

import (
	"config"
	"db"
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
	eventsMaxSize        int
	type_                int
	eventChannels        map[int64]*EventMessagesChannel //eventID -> (userID,MessageChannel)
	userChannels         map[int]*MessageChannel         //userID -> MessageChannel
	waitGroup            *sync.WaitGroup
	config               config.GlobalConfig
	noSqlCollection      *db.MongoCollection
}

func NewRoomFromConfig(waitGroup *sync.WaitGroup, collectionName string) *Room {

	var globalConfig = config.GetConfig()

	return &Room{
		thisPort:             globalConfig.ConnectionsConfig.Client.ListenPort,
		kafkaAddress:         globalConfig.ConnectionsConfig.KafkaServer.ServerUrls,
		clientBuffSize:       globalConfig.ConnectionsConfig.Client.MaxBuffSize,
		eventChannels:        make(map[int64]*EventMessagesChannel, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		userChannels:         make(map[int]*MessageChannel, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		waitGroup:            waitGroup,
		eventsMaxSize:        globalConfig.ConnectionsConfig.Room.EventsMaxSize,
		maxClientConnections: globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize,
		config:               globalConfig,
		noSqlCollection:      db.GetCollection(collectionName),
	}

}
