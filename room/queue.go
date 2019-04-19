package room

import (
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kafka/consumergroup"
	"log"
	"partyfy-message-service/persistient"
	"time"
)

type EventActionRecord struct {
	EventID int64 `json:"eventID"`
}

type UserActionRecord struct {
	ReceiverID int `json:"receiverID"`
}

type EventUserActionRecord struct {
	EventActionRecord
	UserActionRecord
}

type MultipleUserEventActionRecord struct {
	EventActionRecord
	UsersIDs []int `json:"receiversIDs"`
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

	err := errors.New("")
	var cg *consumergroup.ConsumerGroup
	for err != nil {
		cg, err = consumergroup.JoinConsumerGroup(cgroup, topics, zookeeper, kafkaConfiguration)
	}
	log.Print("Connected to queue successfully")

	room.consumeMessagesFromQueue(cg)
	err = cg.Close()
	if err != nil {
		log.Println("Error while closing queue consumer group : ", err)
	}

}

func (room *Room) consumeMessagesFromQueue(cg *consumergroup.ConsumerGroup) {

	log.Println("Starting receiving messages from queue")
	msg := <-cg.Messages()
	err := cg.CommitUpto(msg)
	if err != nil {
		fmt.Println("Error commit zookeeper: ", err.Error())
	}
	for {
		select {
		case msg = <-cg.Messages():
			err := cg.CommitUpto(msg)
			if err != nil {
				fmt.Println("Error commit zookeeper: ", err.Error())
			}
			room.processIncomingRecord(msg)
		}
	}

}

func (room *Room) processIncomingRecord(msg *sarama.ConsumerMessage) {

	topic := msg.Topic
	log.Print("New message from : ", topic)

	switch topic {
	case "event-updated", "event-deleted", "event-created":
		record := unmarshalEventActionRecord(msg.Value)
		if record != nil {
			go room.processEventActionRecord(*record, string(msg.Value), topic)
		}
	case "event-user-removed", "event-user-added":
		record := unmarshalEventUserActionRecord(msg.Value)
		if record != nil {
			go room.processEventUserActionRecord(*record, string(msg.Value), topic)
		}
	case "user-relation-created":
		record := unmarshalUserActionRecord(msg.Value)
		if record != nil {
			go room.processUserActionRecord(*record, string(msg.Value), topic)
		}
	case "image-added", "image-user-attached":
		record := unmarshalImageAddedRecord(msg.Value)
		if record != nil {
			go room.processMultipleUserEventActionRecord(*record, string(msg.Value), topic)
		}
	}
}

func (room *Room) processEventActionRecord(record EventActionRecord, payload, topic string) {
	message := &persistient.EventMessage{
		EventID: record.EventID,
		Channel: topic,
		Body:    payload,
		IsSent:  false,
	}
	room.sendMessageToEventChannel(message)
}

func (room *Room) processUserActionRecord(record UserActionRecord, payload, topic string) {
	message := &persistient.EventMessage{
		ReceiverID: record.ReceiverID,
		Body:       payload,
		Channel:    topic,
		IsSent:     false,
	}
	room.sendMessageToUser(message)
}

func (room *Room) processEventUserActionRecord(record EventUserActionRecord, payload, topic string) {
	message := &persistient.EventMessage{
		ReceiverID: record.ReceiverID,
		EventID:    record.EventID,
		Body:       payload,
		Channel:    topic,
		IsSent:     false,
	}
	room.sendMessageToUser(message)
	room.sendMessageToEventChannel(message)
}

func (room *Room) processMultipleUserEventActionRecord(record MultipleUserEventActionRecord, payload, topic string) {
	message := &persistient.EventMessage{
		Channel: topic,
		Body:    payload,
		IsSent:  false,
	}
	for receiverUserID := range record.UsersIDs {
		message.ReceiverID = receiverUserID
		room.sendMessageToUser(message)
	}
	if record.EventID != 0 {
		message.EventID = record.EventID
		room.sendMessageToEventChannel(message)
	}
}
