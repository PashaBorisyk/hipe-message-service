package room

import (
	"config"
	"container/list"
	"encoding/json"
	"golang.org/x/net/websocket"
	"log"
	"model"
	"net/http"
	"sync"
)

type RoomChannel chan *model.Event

type Room struct {
	thisPort             string
	serverAddr           string
	clientBuffSize       int
	serverBuffSize       int
	maxClientConnections int
	channels             map[int]*RoomChannel
	channelsCount        int
	serverConnection     *websocket.Conn
	waitGroup            *sync.WaitGroup
	serverError          error
	eventsList           *list.List
	eventsMaxSize        int
	config               *config.GlobalConfig
}

func NewRoom(thisAddr, serverAddr string, clientBufSize, serverBufSize int, maxClientSize int, waitGroup *sync.WaitGroup) *Room {

	return &Room{
		thisPort:         thisAddr,
		serverAddr:       serverAddr,
		clientBuffSize:   clientBufSize,
		serverBuffSize:   serverBufSize,
		channels:         make(map[int]*RoomChannel, maxClientSize),
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
		channels:             make(map[int]*RoomChannel, globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize),
		serverConnection:     nil,
		waitGroup:            waitGroup,
		eventsList:           list.New(),
		eventsMaxSize:        globalConfig.ConnectionsConfig.Room.EventsMaxSize,
		maxClientConnections: globalConfig.ConnectionsConfig.Client.MaxConnectionPoolSize,
		config:               &globalConfig,
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

func (room *Room) InitServerConnection(repairCode *chan bool) {
	log.Println("Creating server connection...")

	defer room.waitGroup.Done()

	conn, err := websocket.Dial(room.serverAddr, "ws", room.serverAddr)
	if err != nil {
		log.Println("Unable to connect to server with addres " + room.serverAddr + " protocol : ws")
		log.Println(err)
		return
	}
	room.serverConnection = conn
	room.waitGroup.Add(1)
	go room.serveServerConnection(repairCode)
	log.Println("Server connection" + room.serverAddr + " created successfully")

}

func (room *Room) serveServerConnection(repairCode *chan bool) {
	defer room.waitGroup.Done()

	addr := room.serverConnection.RemoteAddr().String()

	room.serverConnection.MaxPayloadBytes = room.serverBuffSize
	for {
		msg := make([]byte, room.clientBuffSize)
		read, err := room.serverConnection.Read(msg)
		if err != nil {
			log.Println("Closing server connection with addres " + addr)
			log.Println(err)
			room.serverConnection.Close()
			*repairCode <- true
			return
		}

		event := new(model.Event)
		json.Unmarshal(msg[0:read], &event)
		room.addEvent(event)

		for _, v := range room.channels {
			*v <- event
		}

	}

	log.Println("Server connection " + addr + " closed")
	log.Println("Repairing connection")

	*repairCode <- true

}

func (room *Room) serveClientConnection(ws *websocket.Conn) {
	room.waitGroup.Add(1)
	defer room.waitGroup.Done()

	if room.channelsCount >= room.maxClientConnections {
		log.Println("Cannot create connection due the stack is full")
		return
	}

	room.channelsCount++
	channelId := room.channelsCount

	addr := ws.RemoteAddr().String()
	log.Print("Client connection " + addr + " created")

	ws.MaxPayloadBytes = room.clientBuffSize

	for e := room.eventsList.Front(); e != nil; e = e.Next() {
		raw, err := json.Marshal(e)
		if err == nil {
			ws.Write(raw)
		}
	}

	thisChannel := make(RoomChannel)
	room.channels[channelId] = &thisChannel

	for {
		event := <-thisChannel
		raw, err := json.Marshal(event)
		if err == nil {
			ws.Write(raw)
		}
	}

	log.Println("Client connection " + addr + " closed")

	room.channels[channelId] = nil
	room.channelsCount--

}

func (room *Room) addEvent(event *model.Event) {

	if room.eventsList.Len() >= room.eventsMaxSize {
		room.eventsList.Remove(room.eventsList.Front())
	}

	room.eventsList.PushBack(event)

}
