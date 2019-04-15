package room

import (
	"github.com/gorilla/websocket"
	"log"
	"persistient"
	"rest"
)

type MessageChannel chan persistient.EventMessage
type EventMessagesChannel map[int]*MessageChannel

func (room *Room) addUserToEventChannels(messageChannel *MessageChannel, userId int, eventIds ...int64) {

	for _, eventId := range eventIds {

		//add user message channel to EventChannels
		if room.eventChannels[eventId] == nil {

			log.Println("Creating EventMessages pool for event id : ", eventId)

			userChannelsMap := make(EventMessagesChannel, room.maxClientConnections)
			room.eventChannels[eventId] = &userChannelsMap
		}
		event := room.eventChannels[eventId]
		(*event)[userId] = messageChannel

	}

}

func (room *Room) createMessageChannel(userId int) *MessageChannel {
	//create user message channel ad it to MessageChannels and return a pointer
	userMessageChannel := make(MessageChannel)
	room.userChannels[userId] = &userMessageChannel
	return &userMessageChannel
}

func (room *Room) createNewEventChannel(eventID int64) error {

	usersIDs, err := rest.GetUserIDsByEventID(eventID)
	if err != nil {
		log.Print("Unable to get users for created event. eventID: ", eventID)
		return err
	}

	for userID := range usersIDs {
		channel := room.userChannels[userID]
		if channel != nil {
			room.addUserToEventChannels(channel, userID, eventID)
		}
	}
	return nil

}

func (room *Room) sendMessageToUser(message *persistient.EventMessage) {
	userChannel := room.userChannels[message.ReceiverID]
	if userChannel != nil {
		*userChannel <- *message
	}
}

func (room *Room) sendMessageToEventChannel(message *persistient.EventMessage) {

	if room.eventChannels[message.EventID] == nil {
		err := room.createNewEventChannel(message.EventID)
		if err != nil {
			log.Print("Error creating new event channel: ", err)
			return
		}
	}

	userIds, err := rest.GetUserIDsByEventID(message.EventID)
	if err != nil {
		log.Println("Error getting userIds. Notifications would not be sent : ", err)
		return
	}

	usersIdsList := arrayToMap(userIds)

	eventRoom := room.eventChannels[message.EventID]
	if eventRoom == nil {
		log.Println("No event channel found with eventID :", message.EventID)
		return
	}
	for userId, userChannel := range *eventRoom {
		*userChannel <- *message
		(*usersIdsList)[userId] = false
	}
}

func (room *Room) startIncomingClientMessagesRoutine(userConnection *websocket.Conn, userId int) {

	var eventMessage persistient.EventMessage

	for {
		err := userConnection.ReadJSON(&eventMessage)
		log.Println("Incoming message")
		if err != nil {
			log.Println("Error while reading from buffer :", err)
			_ = userConnection.Close()
			break
		}

		eventRoom := room.eventChannels[eventMessage.EventID]
		if eventRoom == nil {
			continue
		}
		for userID, messageChannel := range *eventRoom {
			log.Print("Sending user message to user with ID ", userID)
			*messageChannel <- eventMessage
			log.Print("Message to user with ID", userID, " sent.")
		}

	}
}

func (room *Room) startOutgoingClientMessagesRoutine(userConnection *websocket.Conn, channel *MessageChannel, outputStarted *chan bool, userId int) {
	*outputStarted <- true
	for {
		message := <-(*channel)
		err := userConnection.WriteJSON(message)
		if err != nil {
			log.Println("Error while writing message to client :", err)
			continue
		}

	}

}

func arrayToMap(array []int) *map[int]bool {
	resultMap := make(map[int]bool)
	for value := range array {
		resultMap[value] = true
	}
	return &resultMap
}
