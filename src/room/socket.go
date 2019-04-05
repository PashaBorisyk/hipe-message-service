package room

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"persistient"
	"rest"
	"strconv"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (room *Room) InitClientConnectionsHandler() {
	log.Println("Waiting for client connections")
	defer room.waitGroup.Done()
	http.HandleFunc("/", room.serveClientConnection)
	err := http.ListenAndServe(room.config.ConnectionsConfig.Client.ListenPort, nil)
	if err != nil {
		log.Print("Error while listening to connections")
		log.Fatal(err)
	}
}

func (room *Room) serveClientConnection(w http.ResponseWriter, r *http.Request) {
	userConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil{
		log.Print("Error upgrading request : ", err)
	}

	//log.Println("Creating client connection : " + userConnection.RemoteAddr().String())
	room.waitGroup.Add(1)
	defer room.waitGroup.Done()
	//userConnection.MaxPayloadBytes = room.clientBuffSize

	if room.channelsCount >= room.maxClientConnections {
		log.Println("Cannot create connection due the stack is full")
		return
	}
	room.channelsCount++
	addr := userConnection.RemoteAddr().String()
	log.Print("Client connection from " + addr + " created")

	userID, err := getUserID(r)
	if err != nil {
		log.Print("Can not process user connection without userID")
		return
	}

	eventIds, err := rest.GetEventsIDsMemberID(userID)

	messageChannel := room.createMessageChannel(userID)
	room.addUserToEventChannels(messageChannel, userID, eventIds...)
	room.clientCounter++

	ioChannel := make(MessageChannel)
	outputStarted := make(chan bool)

	log.Println("Client connection " + userConnection.RemoteAddr().String() + " successfully instantiated; Users online : " + strconv.Itoa(room.clientCounter))
	go room.processServerOutput(userConnection, &ioChannel, &outputStarted, userID)
	<-outputStarted
	room.processServerInput(userConnection, userID)

	room.clientCounter--
	log.Println("Closing client connection " + userConnection.RemoteAddr().String() + " ; Users online : " + strconv.Itoa(room.clientCounter))

}

func (room *Room) processServerOutput(userConnection *websocket.Conn, chanel *MessageChannel, outputStarted *chan bool, userId int) {
	*outputStarted <- true

	for {
		message := <-*chanel
		err := userConnection.WriteJSON(message)
		if err != nil {
			log.Println("Error while writing message to client")
			break
		}

	}

}

func (room *Room) processServerInput(userConnection *websocket.Conn, userId int) {

	var eventMessage persistient.EventMessage

	for {
		err := userConnection.ReadJSON(&eventMessage)
		log.Println("Incoming message")
		if err != nil {
			log.Println("Error while reading from buffer")
			log.Println(err)
			break

		}

		eventRoom := room.eventChannels[eventMessage.EventID]
		for _, messageChannel := range *eventRoom {
			*messageChannel <- eventMessage
		}

	}
}

func getUserID(r *http.Request) (int, error) {
	query := r.URL.Query()
	userID, err := strconv.ParseInt(query.Get(UserID), 10, 32)

	if err != nil {
		log.Println("Cannot parse id from query string  : "+query.Encode(), " ", err)
		return 0, err
	}
	return int(userID), nil
}
