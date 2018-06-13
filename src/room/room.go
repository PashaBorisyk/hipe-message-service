package room

import (
	"config"
	"encoding/json"
	"golang.org/x/net/websocket"
	"log"
	"model"
	"net/http"
	"sync"
	"strconv"
	"io/ioutil"
)

const GetEventsPath = "/event/get_by_member_id/"
const UserId = "user_id"

type Room struct {
	serverError          error
	thisPort             string
	serverAddr           string
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
	serverConnection     map[config.Type]*websocket.Conn
	config               config.GlobalConfig
}

func NewRoom(thisAddr, serverAddr string, clientBufSize, serverBufSize int, maxClientSize int, waitGroup *sync.WaitGroup) *Room {

	return &Room{
		thisPort:         thisAddr,
		serverAddr:       serverAddr,
		clientBuffSize:   clientBufSize,
		serverBuffSize:   serverBufSize,
		eventChannels:    make(map[int64]*model.EventBasket, maxClientSize),
		userChannels:     make(map[int64]*model.MessageChannel, maxClientSize),
		serverConnection: nil,
		waitGroup:        waitGroup,
	}

}

func NewRoomFromConfig(waitGroup *sync.WaitGroup) *Room {

	var globalConfig = config.GetConfig()

	return &Room{
		thisPort:             globalConfig.ConnectionsConfig.Client.ListenPort,
		serverAddr:           globalConfig.ConnectionsConfig.Server.Url,
		clientBuffSize:       globalConfig.ConnectionsConfig.Client.MaxBuffSize,
		eventChannels:        make(map[int64]*model.EventBasket, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		userChannels:         make(map[int64]*model.MessageChannel, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		serverConnection:     make(map[config.Type]*websocket.Conn),
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

func (room *Room) InitServerConnection() {
	log.Println("Creating server connection...")

	defer room.waitGroup.Done()

	for _, connectionType := range room.config.ConnectionsConfig.Server.Types {

		query := room.serverAddr + "?type=" + strconv.Itoa(connectionType.Type) + "&category=" + strconv.Itoa(connectionType.Category)

		conn, err := websocket.Dial(room.serverAddr, "ws", query)
		if err != nil {
			log.Println("Unable to connect to server with addres " + room.serverAddr + " protocol : ws")
			log.Println(err)
			return
		}
		room.serverConnection[connectionType] = conn
		room.waitGroup.Add(1)
		go room.serveServerConnection(&connectionType, conn)
	}

	log.Println("Server connection" + room.serverAddr + " created successfully")

}

func (room *Room) serveServerConnection(connectionType *config.Type, connection *websocket.Conn) {
	defer room.waitGroup.Done()

	addr := connection.RemoteAddr().String()

	connection.MaxPayloadBytes = room.serverBuffSize
	for {
		msg := make([]byte, room.clientBuffSize)
		read, err := connection.Read(msg)
		if err != nil {
			log.Println("Closing server connection with addres " + addr)
			log.Println(err)
			connection.Close()
			return
		}

		event := new(model.EventMessage)
		json.Unmarshal(msg[0:read], &event)

	}

	log.Println("Server connection " + addr + " closed")
	log.Println("Repairing connection")

}

func (room *Room) serveClientConnection(ws *websocket.Conn) {
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

	userId := ws.Request().Header.Get(UserId)

	getEventsUrl := room.config.ConnectionsConfig.Server.Url + GetEventsPath + userId

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
		log.Println("Unable to parse json : " + string(body))
	}

	thisChannel := make(model.MessageChannel)

	inputStarted := make(chan bool)
	outputStarted := make(chan bool)

	userID, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		log.Println("User id is not an int : " + userId)
	}

	go room.serveInput(ws, userID, &inputStarted)
	go room.serveOutput(ws,&thisChannel,&outputStarted)

	<-inputStarted
	<-outputStarted

}

func (room *Room) serveInput(ws *websocket.Conn, userId int64, inputClosed *chan bool) {
	*inputClosed <- true

	payload := make([]byte, room.config.ConnectionsConfig.Client.MaxBuffSize)
	var eventMessage model.EventMessage

	for {
		read, err := ws.Read(payload)
		if err == nil {
			err = json.Unmarshal(payload[0:read], &eventMessage)
			if err != nil {
				basket := room.eventChannels[eventMessage.EventId]
				for k, v := range *basket {
					if userId != k {
						*v <- eventMessage
					}
				}

			}
		}
	}

}

func (room *Room) serveOutput(ws *websocket.Conn,chanel *model.MessageChannel , outputClosed *chan bool) {
	*outputClosed <- true

	for {
		message := <- *chanel
		payload, err := json.Marshal(message)
		if err == nil {
		 	_,err := ws.Write(payload)
			if err != nil {
				log.Println("Error while writing message to client")
			}
		}
	}

}
