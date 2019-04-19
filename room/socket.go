package room

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
)

var upgrade = websocket.Upgrader{
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

	room.waitGroup.Add(1)
	defer room.waitGroup.Done()

	userConnection, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Error upgrading request : ", err)
	}
	defer userConnection.Close()

	//log.Println("Creating client connection : " + userConnection.RemoteAddr().String())
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

	messageChannel := room.createMessageChannel(userID)
	room.clientCounter++

	//userMessageChannel := make(MessageChannel,10)
	outputStarted := make(chan bool)

	log.Println("Client connection " + userConnection.RemoteAddr().String() + " successfully instantiated; Users online : " + strconv.Itoa(len(room.userChannels)))
	go room.startOutgoingClientMessagesRoutine(userConnection, messageChannel, &outputStarted, userID)
	<-outputStarted
	room.startIncomingClientMessagesRoutine(userConnection, userID)

	log.Println("Closing client connection " + userConnection.RemoteAddr().String() + " ; Users online : " + strconv.Itoa(len(room.userChannels)))

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
