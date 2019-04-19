package room

import (
	"github.com/gorilla/websocket"
	"log"
	"partyfy-message-service/db"
	"partyfy-message-service/persistient"
	"partyfy-message-service/rest"
)

type MessageChannel chan persistient.EventMessage
type EventMessagesChannel map[int]*MessageChannel

func (room *Room) createMessageChannel(userId int) *MessageChannel {
	//create user message channel ad it to MessageChannels and return a pointer
	userMessageChannel := make(MessageChannel)
	room.userChannels[userId] = &userMessageChannel
	return &userMessageChannel
}

func (room *Room) sendMessageToUser(message *persistient.EventMessage) {

	messagesCollection := db.GetCollection(db.MessagesCollection)

	userChannel := room.userChannels[message.ReceiverID]
	if userChannel != nil {
		*userChannel <- *message
	} else {
		_ = messagesCollection.Insert(*message)
	}
}

func (room *Room) sendMessageToEventChannel(message *persistient.EventMessage) {

	messagesCollection := db.GetCollection(db.MessagesCollection)

	userIds, err := rest.GetUserIDsByEventID(message.EventID)
	if err != nil {
		log.Println("Error getting userIds. Notifications would not be sent : ", err)
		_ = messagesCollection.Insert(*message)
		return
	}

	for userId := range userIds {
		message.ReceiverID = userId
		room.sendMessageToUser(message)
	}

}

func (room *Room) startIncomingClientMessagesRoutine(userConnection *websocket.Conn, userID int) {

	defer userConnection.Close()
	defer delete(room.userChannels, userID)
	var eventMessage persistient.EventMessage

	for {
		err := userConnection.ReadJSON(&eventMessage)
		if err != nil {
			log.Println("Error while reading from buffer :", err)
			break
		}

		if eventMessage.ReceiverID != 0 && eventMessage.EventID != 0 {
			errorMsg := persistient.EventMessage{ReceiverID: userID, Channel: "error", Body: "Unable to send message both to event and user"}
			err = sendMessageToSocketConnection(userConnection, &errorMsg)
			if err != nil {
				log.Println("Error sending error message")
				break
			}
			continue
		}

		eventMessage.SenderID = userID
		eventMessage.IsSent = false
		eventMessage.Channel = "message"

		if eventMessage.ReceiverID != 0 {
			room.sendMessageToUser(&eventMessage)
		} else if eventMessage.EventID != 0 {
			room.sendMessageToEventChannel(&eventMessage)
		}

	}

}

func (room *Room) startOutgoingClientMessagesRoutine(clientConnection *websocket.Conn, channel *MessageChannel, outputStarted *chan bool, userID int) {
	*outputStarted <- true
	defer clientConnection.Close()
	defer delete(room.userChannels, userID)

	messagesCollection := db.GetCollection("messages")
	err := sendUnsentMessages(clientConnection, userID)
	if err != nil {
		log.Print("Error sending unsent messages. Maybe you should check database")
	}
	for {
		message := <-(*channel)
		if message.SenderID == userID {
			continue
		}
		err := sendMessageToSocketConnection(clientConnection, &message)
		if err != nil {
			log.Print("Error with client connection occurred. Closing connection")
			_ = messagesCollection.Insert(message)
			break
		}
	}

}

func sendMessageToSocketConnection(connection *websocket.Conn, message *persistient.EventMessage) error {

	messagesCollection := db.GetCollection(db.MessagesCollection)
	err := connection.WriteJSON(message)
	if err != nil {
		log.Println("Error while writing message to client :", err)
		return err
	} else {
		message.IsSent = true
		_ = messagesCollection.Insert(*message)
		return nil
	}
}

func sendUnsentMessages(connection *websocket.Conn, userID int) error {
	messagesCollection := db.GetCollection("messages")
	err := messagesCollection.FindUnsentByReceiverUserID(userID, func(message persistient.EventMessage, err error) error {
		if err != nil {
			log.Println("Error while unmarshal EventMessage")
			return err
		}
		err = sendMessageToSocketConnection(connection, &message)
		if err != nil {
			log.Print("Error sending unsent message: ", err)
			return err
		}
		messagesCollection.SetMessageSent(message.ID)
		return nil
	})
	if err != nil {
		log.Print("Unable to get user unsent messages: ", err)
		return err
	}
	return nil
}
