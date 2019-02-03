package room

import (
	"config"
	"db"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kafka/consumergroup"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"log"
	"models"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	GetEventsPath = "/event/get_by_member_id/"
	UserId        = "user_id"
	CREATED       = "CREATED"
	UPDATED       = "UPDATED"
	DELETED       = "DELETED"
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
	eventChannels        map[int64]*models.EventMessagesChannel
	//Keeps user connections k -> UserId, v-> Channel which recieves EventMessage
	userChannels map[int64]*models.MessageChannel
	waitGroup    *sync.WaitGroup
	config       config.GlobalConfig
	collection   *db.MongoCollection
}

//func NewRoom(thisAddr, kafkaAddress string, clientBufSize, serverBufSize int, maxClientSize int, waitGroup *sync.WaitGroup) *Room {
//
//	return &Room{
//		thisPort:         thisAddr,
//		kafkaAddress:     kafkaAddress,
//		clientBuffSize:   clientBufSize,
//		serverBuffSize:   serverBufSize,
//		eventChannels:    make(map[int64]*models.EventMessagesChannel, maxClientSize),
//		userChannels:     make(map[int64]*models.MessageChannel, maxClientSize),
//		serverConnection: nil,
//		waitGroup:        waitGroup,
//	}
//
//}

func NewRoomFromConfig(waitGroup *sync.WaitGroup, collectionName string) *Room {

	var globalConfig = config.GetConfig()

	return &Room{
		thisPort:             globalConfig.ConnectionsConfig.Client.ListenPort,
		kafkaAddress:         globalConfig.ConnectionsConfig.KafkaServer.ServerUrls,
		clientBuffSize:       globalConfig.ConnectionsConfig.Client.MaxBuffSize,
		eventChannels:        make(map[int64]*models.EventMessagesChannel, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		userChannels:         make(map[int64]*models.MessageChannel, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		waitGroup:            waitGroup,
		eventsMaxSize:        globalConfig.ConnectionsConfig.Room.EventsMaxSize,
		maxClientConnections: globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize,
		config:               globalConfig,
		collection:           db.GetCollection(collectionName),
	}

}

func (room *Room) InitClientConnectionsHandler() {
	log.Println("Waiting for incommeng client connections")
	defer room.waitGroup.Done()
	http.Handle("/", websocket.Handler(room.serveClientConnection))
	err := http.ListenAndServe(room.config.ConnectionsConfig.Client.ListenPort, nil)
	if err != nil {
		log.Print("Error while listening to connections")
		log.Fatal(err)
	}
}

func (room *Room) InitKafkaConnection() {
	log.Println("Creating server connection...")

	defer room.waitGroup.Done()

	kafkaConfiguration := consumergroup.NewConfig()
	kafkaConfiguration.Offsets.Initial = sarama.OffsetOldest
	kafkaConfiguration.Offsets.ProcessingTimeout = 10 * time.Second

	cgroup := room.config.ConnectionsConfig.KafkaServer.CGroup
	topics := room.config.ConnectionsConfig.KafkaServer.Topics
	zookeeper := room.config.ConnectionsConfig.KafkaServer.ServerUrls

	cg, err := consumergroup.JoinConsumerGroup(cgroup, topics, zookeeper, kafkaConfiguration)

	if err != nil {
		log.Fatal("Error while connecting zookeeper ", err.Error())
	}
	room.consume(cg)
	err = cg.Close()
	if err != nil {
		log.Println("Error while closing kafka consumer group : ", err)
	}
}

func (room *Room) consume(cg *consumergroup.ConsumerGroup) {

	log.Println("Starting consume")

	eventMessage := new(models.EventMessage)

	for {
		select {
		case msg := <-cg.Messages():
			log.Println("Message got : ", string(msg.Value))
			err := cg.CommitUpto(msg)
			if err != nil {
				fmt.Println("Error commit zookeeper: ", err.Error())
				break
			}

			err = json.Unmarshal(msg.Value, eventMessage)
			if err != nil {
				log.Println(
					"Error while unmarshal json to models.Event. Json string is : ", string(msg.Value), " ", err)
				continue
			}

			log.Println(eventMessage)

			eventRoom := room.eventChannels[eventMessage.EventID]
			if eventRoom != nil {
				for _, messageChannel := range *eventRoom {
					*(messageChannel) <- *eventMessage
				}
			}
		}
	}

	log.Println("Exiting consume")

}

func (room *Room) serveClientConnection(ws *websocket.Conn) {
	log.Println("Creating client connection : " + ws.RemoteAddr().String())
	room.waitGroup.Add(1)
	defer room.waitGroup.Done()
	ws.MaxPayloadBytes = room.clientBuffSize

	if room.channelsCount >= room.maxClientConnections {
		log.Println("Cannot create connection due the stack is full")
		return
	}
	room.channelsCount++
	addr := ws.RemoteAddr().String()
	log.Print("Client connection from " + addr + " created")

	query := ws.Request().URL.Query()
	userId, err := strconv.ParseInt(query.Get(UserId), 10, 64)
	if err != nil {
		log.Println("Cannot parse id from query string  : "+query.Encode(), " ", err)
	}

	getEventsUrl := room.config.ConnectionsConfig.EventProcessor.Url + GetEventsPath + strconv.FormatInt(userId, 10)
	resp, err := http.Get(getEventsUrl)

	if err != nil {
		log.Println("Unable to get users event ids; URL : "+getEventsUrl, " ", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Unable to read event ids array; URL : "+getEventsUrl, " ", err)
	}

	err = resp.Body.Close()
	if err != nil {
		log.Println("Unable to close response body ", err)
	}

	var eventIds []int64
	err = json.Unmarshal(body, &eventIds)

	if err != nil {
		log.Println("Unable to parse event json : "+string(body), " ", err)
		eventIds = []int64{1, 2, 3, 4, 5}
	}

	messageChannel := room.addUserToMessageChannels(userId)
	room.addUserToEventChannels(messageChannel, userId, eventIds...)
	room.clientCounter++

	ioChannel := make(models.MessageChannel)
	outputStarted := make(chan bool)

	log.Println("Client connection " + ws.RemoteAddr().String() + " successfully instantiated; Users online : " + strconv.Itoa(room.clientCounter))
	go room.processServerOutput(ws, &ioChannel, &outputStarted, userId)
	<-outputStarted
	room.processServerInput(ws, userId)

	room.clientCounter--
	log.Println("Closing client connection " + ws.RemoteAddr().String() + " ; Users online : " + strconv.Itoa(room.clientCounter))

}

func (room *Room) processServerInput(ws *websocket.Conn, userId int64) {

	payload := make([]byte, room.config.ConnectionsConfig.Client.MaxBuffSize)
	var eventMessage models.EventMessage

	for {
		read, err := ws.Read(payload)
		log.Println("Incomming message ")
		if err != nil {
			log.Println("Error while reading from buffer")
			log.Println(err)
			break

		}

		err = json.Unmarshal(payload[0:read], &eventMessage)
		if err != nil {
			log.Println("Error while parsing Message json : " + string(payload[0:read]))
			log.Println(err)
			break

		} else {
			eventRoom := room.eventChannels[eventMessage.EventID]
			for _, messageChannel := range *eventRoom {
				*messageChannel <- eventMessage
			}
		}

	}
}

func (room *Room) processServerOutput(ws *websocket.Conn, chanel *models.MessageChannel, outputStarted *chan bool, userId int64) {
	*outputStarted <- true

	for {
		message := <-*chanel
		payload, err := json.Marshal(message)
		if err == nil {
			_, err := ws.Write(payload)
			if err != nil {
				log.Println("Error while writing message to client")
				break
			}
		} else {
			break
		}
	}

}

func (room *Room) addUserToEventChannels(messageChannel *models.MessageChannel, userId int64, eventIds ...int64) {

	for _, eventId := range eventIds {

		//add user message channel to EventChannels
		if room.eventChannels[eventId] == nil {

			log.Println("Creating EventMessages pool for event id : ", eventId)

			userChannelsMap := make(models.EventMessagesChannel, room.maxClientConnections)
			room.eventChannels[eventId] = &userChannelsMap
		}
		event := room.eventChannels[eventId]
		(*event)[userId] = messageChannel

	}

}

func (room *Room) addUserToMessageChannels(userId int64) *models.MessageChannel {
	//create user message channel ad it to MessageChannels and return a pointer
	userMessageChannel := make(models.MessageChannel)
	room.userChannels[userId] = &userMessageChannel
	return &userMessageChannel
}
