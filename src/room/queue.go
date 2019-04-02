package room

import (
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kafka/consumergroup"
	"log"
	"models"
	"time"
)

type RecordsHolder struct {
	EventCreatedRecord     *EventCreatedRecord
	ImageAddedRecord       *ImageAddedRecord
	EventUserAddedRecord   *EventUserAddedRecord
	EventUserRemovedRecord *EventUserRemovedRecord
	EventUpdatedRecord     *EventUpdatedRecord
}

type Record struct{}

type UserRelationCreatedRecord struct {
	Record
	UserID       int    `json:"userId"`
	Username     string `json:"username"`
	ReceiverUser int    `json:"receiverUser"`
	RelationType string `json:"relationType"`
	Time         int64  `json:"time"`
}

type EventCreatedRecord struct {
	Record
	UserID   int    `json:"userId"`
	Username string `json:"username"`
	EventID  int64    `json:"eventId"`
	UsersID  []int  `json:"usersId"`
	Time     int64  `json:"time"`
}

type EventUpdatedRecord struct {
	UserID   int64    `json:"userId"`
	Username string `json:"username"`
	EventID  int64    `json:"eventId"`
	Time     int64  `json:"time"`
}

type EventDeletedRecord struct {
	Record
	UserID   int    `json:"userId"`
	Username string `json:"username"`
	EventID  int    `json:"eventId"`
	Time     int64  `json:"time"`
}

type ImageAddedRecord struct {
	Record
	UserID      int    `json:"userId"`
	Username    string `json:"username"`
	ImageID     int    `json:"imageId"`
	EventID     int64    `json:"eventId"`
	MarkedUsers []int  `json:"markedUsers"`
	Time        int64  `json:"time"`
}

type EventUserAddedRecord struct {
	Record
	UserID        int    `json:"userId"`
	Username      string `json:"username"`
	EventID       int64    `json:"eventId"`
	PassiveUserId int    `json:"passiveUserId"`
	Time          int64  `json:"time"`
}

type EventUserRemovedRecord struct {
	Record
	UserID   int    `json:"userId"`
	Username string `json:"username"`
	EventID  int64    `json:"eventId"`
	By       int    `json:"by"`
	Time     int64  `json:"time"`
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
	room.consumeMessagesFromQueue(cg)
	err = cg.Close()
	if err != nil {
		log.Println("Error while closing kafka consumer group : ", err)
	}
}

func (room *Room) consumeMessagesFromQueue(cg *consumergroup.ConsumerGroup) {

	log.Println("Starting consumeMessagesFromQueue")

	for {
		select {
		case msg := <-cg.Messages():
			err := cg.CommitUpto(msg)
			if err != nil {
				fmt.Println("Error commit zookeeper: ", err.Error())
				break
			}

			switch msg.Topic {
			case "event-updated":
				record := unmarshalEventUpdatedRecord(msg.Value)
				go room.processEventUpdatedRecord(*record)
			case "event-deleted":
				record := unmarshalEventDeletedRecord(msg.Value)
				go room.processEventUpdatedRecord(*record)
			case "event-created":
				record := unmarshalEventCreatedRecord(msg.Value)
				go room.processEventUpdatedRecord(*record)
			case "event-user-removed":
				record := new(EventUpdatedRecord)
				err := json.Unmarshal(msg.Value, record)
				if err != nil {
					log.Println("Unable to unmarshall ", string(msg.Value), "to EventUpdatedRecord")
					log.Println(err)
				}
				go room.processEventUpdatedRecord(*record)
			case "event-user-added":
				record := new(EventUpdatedRecord)
				err := json.Unmarshal(msg.Value, record)
				if err != nil {
					log.Println("Unable to unmarshall ", string(msg.Value), "to EventUpdatedRecord")
					log.Println(err)
				}
				go room.processEventUpdatedRecord(*record)
			case "image-added":
				record := new(EventUpdatedRecord)
				err := json.Unmarshal(msg.Value, record)
				if err != nil {
					log.Println("Unable to unmarshall ", string(msg.Value), "to EventUpdatedRecord")
					log.Println(err)
				}
				go room.processEventUpdatedRecord(*record)
			case "image-user-attached":
				record := new(EventUpdatedRecord)
				err := json.Unmarshal(msg.Value, record)
				if err != nil {
					log.Println("Unable to unmarshall ", string(msg.Value), "to EventUpdatedRecord")
					log.Println(err)
				}
				go room.processEventUpdatedRecord(*record)
			case "user-relation-created":
				record := new(EventUpdatedRecord)
				err := json.Unmarshal(msg.Value, record)
				if err != nil {
					log.Println("Unable to unmarshall ", string(msg.Value), "to EventUpdatedRecord")
					log.Println(err)
				}
				go room.processEventUpdatedRecord(*record)
			}

		}
	}

	log.Println("Exiting consumeMessagesFromQueue")
}

func unmarshalEventUpdatedRecord(raw []byte) (record *EventUpdatedRecord) {
	err := json.Unmarshal(raw, record)
	if err != nil {
		log.Println("Unable to unmarshall ", string(raw), "to EventUpdatedRecord")
		log.Println(err)
	}
	return record
}
func unmarshalEventDeletedRecord(raw []byte) (record *EventDeletedRecord) {
	err := json.Unmarshal(raw, record)
	if err != nil {
		log.Println("Unable to unmarshall ", string(raw), "to EventUpdatedRecord")
		log.Println(err)
	}
	return record
}
func unmarshalEventCreatedRecord(raw []byte) (record *EventCreatedRecord) {
	err := json.Unmarshal(raw, record)
	if err != nil {
		log.Println("Unable to unmarshall ", string(raw), "to EventCreatedRecord")
		log.Println(err)
	}
	return record
}

func (room *Room) processEventUpdatedRecord(record EventUpdatedRecord,topic string) {



	message := models.EventMessage{
		EventID:record.EventID,
		Time:record.Time,
		Channel:topic,
	}

	eventRoom := room.eventChannels[record.EventID]
	for userId,userChannel := range *eventRoom {
		userChannel
	}
}
