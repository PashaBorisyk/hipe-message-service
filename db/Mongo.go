package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"partyfy-message-service/config"
	"partyfy-message-service/persistient"
	"time"
)

const MessagesCollection = "messages"

var database *mongo.Database
var collectionsMap map[string]*Collection

type Collection struct {
	collection *mongo.Collection
}

func init() {

	mongoConfig := config.GetConfig().ConnectionsConfig.Mongo
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoConfig.Uri))
	if err != nil {
		log.Fatal("Unable to create Mongo client with address: "+mongoConfig.Uri, "; ", err)
		return
	}

	ctx := createContext()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal("Unable to connect to Mongo database with address: ", mongoConfig.Uri, "; ", err)
		return
	}

	database = client.Database(mongoConfig.Database)
	collectionsMap = make(map[string]*Collection)
}

func GetCollection(name string) *Collection {
	if collectionsMap[name] == nil {
		collectionsMap[name] = &Collection{database.Collection(name)}
	}
	return collectionsMap[name]
}

func (holder *Collection) Insert(docs ...interface{}) error {
	ctx := createContext()
	_, err := holder.collection.InsertMany(ctx, docs)
	if err != nil {
		log.Println("Error inserting document: ", err)
	}
	return err
}

func (holder *Collection) FindByReceiverID(userID int, foreach func(message persistient.EventMessage, err error) error) error {
	ctx := createContext()
	queryResult, err := holder.collection.Find(ctx, bson.M{"receiverID": userID})
	if err != nil {
		log.Println("Unable to get result from FindUnsentByReceiverUserID: ", err)
	}
	defer queryResult.Close(ctx)
	return decodeMultipleResult(queryResult, foreach)
}

func (holder *Collection) FindUnsentByReceiverUserID(userID int, foreach func(message persistient.EventMessage, err error) error) error {
	ctx := createContext()
	queryResult, err := holder.collection.Find(ctx, bson.M{"isSent": false, "receiverID": userID})
	if err != nil {
		log.Println("Unable to get result from FindUnsentByReceiverUserID: ", err)
	}
	defer queryResult.Close(ctx)
	return decodeMultipleResult(queryResult, foreach)
}

func (holder *Collection) FindMessagesForEvent(eventID int64, foreach func(message persistient.EventMessage, err error) error) error {
	ctx := createContext()
	queryResult, err := holder.collection.Find(ctx, bson.M{"eventID": eventID})
	if err != nil {
		log.Println("Unable to get result from FindUnsentByReceiverUserID: ", err)
	}
	defer queryResult.Close(ctx)
	return decodeMultipleResult(queryResult, foreach)
}

func (holder *Collection) SetMessageSent(msgID interface{}) {
	ctx := createContext()
	result := holder.collection.FindOneAndUpdate(ctx,
		bson.M{"_id": msgID},
		bson.D{{"$set", bson.D{{"isSent", true}}}})
	_, err := result.DecodeBytes()
	if err != nil {
		log.Println("Error updating EventMessage document: ", err)
	}
}

func decodeMultipleResult(cursor *mongo.Cursor, foreach func(message persistient.EventMessage, err error) error) error {
	ctx := createContext()
	var message persistient.EventMessage
	for cursor.Next(ctx) {
		err := cursor.Decode(&message)
		if err != nil {
			log.Println("Unable to decode document: ", err)
		}
		message.ID = cursor.Current.Lookup("_id").ObjectID()
		if foreach(message, err) != nil {
			break
		}
	}
	cursor.Close(ctx)
	err := cursor.Err()
	if err != nil {
		log.Println(err)
	}
	return err
}

func createContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	return ctx
}
