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
	clientCounter        int
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
		serverAddr:           globalConfig.ConnectionsConfig.Server.WsUrl,
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

	repairer := make(chan *config.Type)
	for _, connectionType := range room.config.ConnectionsConfig.Server.Types {
		room.createServerConnection(connectionType, &repairer)
	}

	for {
		connType := <-repairer
		if connType.Type == -1 || connType.Category == -1 {
			break
		}
		room.createServerConnection(*connType, &repairer)
	}

}

func (room *Room) createServerConnection(connectionType config.Type, repairer *chan *config.Type) {

	addr := room.serverAddr + "?for_type=" + strconv.Itoa(connectionType.Type) + "&for_category=" + strconv.Itoa(connectionType.Category)
	origin := "http://*"
	conn, err := websocket.Dial(addr, "ws", origin)
	if err != nil {
		log.Println("Unable to connect to server with addres " + room.serverAddr + " protocol : ws")
		log.Println(err)
		return
	}
	room.serverConnection[connectionType] = conn
	room.waitGroup.Add(1)
	go room.serveServerConnection(connectionType, conn, repairer)
	log.Println("Server connection " + addr + " created successfully")

}

func (room *Room) serveServerConnection(connectionType config.Type, connection *websocket.Conn, repairer *chan *config.Type) {
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
			break
		}

		event := new(model.EventMessage)
		json.Unmarshal(msg[0:read], &event)

	}

	log.Println("Server connection " + addr + " closed")
	log.Println("Repairing connection")
	*repairer <- &connectionType

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

	getEventsUrl := room.config.ConnectionsConfig.Server.HttpUrl + GetEventsPath + userId

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
