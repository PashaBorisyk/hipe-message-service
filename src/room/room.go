package room

import (
	"config"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kafka/consumergroup"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"log"
	"model"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const GetEventsPath = "/event/get_by_member_id/"
const UserId = "user_id"

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
	category             int
	eventChannels        map[int64]*model.EventBasket
	userChannels         map[int64]*model.MessageChannel
	waitGroup            *sync.WaitGroup
	serverConnection     map[config.Topics]*websocket.Conn
	config               config.GlobalConfig
}

//func NewRoom(thisAddr, kafkaAddress string, clientBufSize, serverBufSize int, maxClientSize int, waitGroup *sync.WaitGroup) *Room {
//
//	return &Room{
//		thisPort:         thisAddr,
//		kafkaAddress:     kafkaAddress,
//		clientBuffSize:   clientBufSize,
//		serverBuffSize:   serverBufSize,
//		eventChannels:    make(map[int64]*model.EventBasket, maxClientSize),
//		userChannels:     make(map[int64]*model.MessageChannel, maxClientSize),
//		serverConnection: nil,
//		waitGroup:        waitGroup,
//	}
//
//}

func NewRoomFromConfig(waitGroup *sync.WaitGroup) *Room {

	var globalConfig = config.GetConfig()

	return &Room{
		thisPort:             globalConfig.ConnectionsConfig.Client.ListenPort,
		kafkaAddress:         globalConfig.ConnectionsConfig.KafkaServer.ServerUrls,
		clientBuffSize:       globalConfig.ConnectionsConfig.Client.MaxBuffSize,
		eventChannels:        make(map[int64]*model.EventBasket, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		userChannels:         make(map[int64]*model.MessageChannel, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		serverConnection:     make(map[config.Topics]*websocket.Conn),
		waitGroup:            waitGroup,
		eventsMaxSize:        globalConfig.ConnectionsConfig.Room.EventsMaxSize,
		maxClientConnections: globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize,
		config:               globalConfig,
		category:             globalConfig.ConnectionsConfig.Room.Category,
	}

}

func (room *Room) InitClientConnections() {
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
	defer cg.Close()
	if err != nil {
		log.Fatal("Error while connecting zookeeper ", err.Error())
	}
	consume(cg)
}

func consume(cg *consumergroup.ConsumerGroup) {

	for {
		select {
		case msg := <-cg.Messages():
			log.Println("Message got : ", string(msg.Value))
			err := cg.CommitUpto(msg)
			if err != nil {
				fmt.Println("Error commit zookeeper: ", err.Error())
			}

		}
	}

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
	log.Print("Client connection " + addr + " created")

	userId := ws.Request().URL.Query().Get(UserId)

	getEventsUrl := room.config.ConnectionsConfig.EventProcessor.Url + GetEventsPath + userId

	resp, err := http.Get(getEventsUrl)
	if err != nil {
		log.Println("Unable to get users event ids; URL : " + getEventsUrl)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Println("Unable to read event ids array; URL : " + getEventsUrl)
	}
	var eventIds []int64

	err = json.Unmarshal(body, &eventIds)

	if err != nil {
		log.Println("Unable to parse event json : " + string(body))
		eventIds = []int64{1, 2, 3, 4, 5}
	}

	thisChannel := make(model.MessageChannel)

	outputStarted := make(chan bool)

	userID, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		log.Println("User id is not an int : " + userId)
	}

	room.clientCounter++
	log.Println("Client connection " + ws.RemoteAddr().String() + " successfully instatieted; Users online : " + strconv.Itoa(room.clientCounter))
	go room.serveOutput(ws, &thisChannel, &outputStarted)
	<-outputStarted
	room.serveInput(ws, userID)

	room.clientCounter--
	log.Println("Closing client connection " + ws.RemoteAddr().String() + " ; Users online : " + strconv.Itoa(room.clientCounter))

}

func (room *Room) serveInput(ws *websocket.Conn, userId int64) {

	payload := make([]byte, room.config.ConnectionsConfig.Client.MaxBuffSize)
	var eventMessage model.EventMessage

	for {
		read, err := ws.Read(payload)
		log.Println("Incomming message ")
		if err == nil {
			err = json.Unmarshal(payload[0:read], &eventMessage)
			if err == nil {
				basket := room.eventChannels[eventMessage.EventId]
				for k, v := range *basket {
					if userId != k {
						*v <- eventMessage
					}
				}
			} else {
				log.Println("Error while parsing Message json : " + string(payload[0:read]))
				log.Println(err)
				break
			}
		} else {
			log.Println("Error while reading from buffer")
			log.Println(err)
			break
		}
	}
}

func (room *Room) serveOutput(ws *websocket.Conn, chanel *model.MessageChannel, outputStarted *chan bool) {
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
