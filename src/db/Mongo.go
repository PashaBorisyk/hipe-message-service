package db

import (
	"config"
	"gopkg.in/mgo.v2"
	"log"
	"models"
)

var database *mgo.Database

type MongoCollection struct {
	collection *mgo.Collection
}

func my() {

	mongoConfig := config.GetConfig().ConnectionsConfig.Mongo
	session, err := mgo.Dial(mongoConfig.Uri)
	if err != nil {
		log.Print("Unable to connect to Mongo db with address: " + mongoConfig.Uri)
		log.Print(err)
		return
	}

	databaseNames, err := session.DatabaseNames()
	if err != nil {
		log.Print("Error occured while getting database names")
		log.Print(err)
		return
	}

	hasName := false

	for _, name := range databaseNames {
		if name == mongoConfig.Database {
			hasName = true
			break
		}
	}

	if !hasName {
		log.Print("No such database found : " + mongoConfig.Database)
		return
	}

	hasName = false

	database = session.DB(mongoConfig.Database)

	collectionsNames, err := database.CollectionNames()
	if err != nil {
		log.Print("Error occured while getting collections names")
		log.Print(err)
		return
	}

	checkCollectionsAvailibility(mongoConfig.Collections, collectionsNames)

}

func GetCollection(name string) *MongoCollection {
	return nil
	//return &MongoCollection{
	//	collection: database.C(name),
	//}
}

func checkCollectionsAvailibility(configNames, mongoCollections []string) {

	containsCollection := false

	for _, requiredName := range configNames {

		for _, existingName := range mongoCollections {
			if existingName == requiredName {
				containsCollection = true
				break
			}

			if !containsCollection {
				log.Println("Warning! DataBase does not contains collection : ", requiredName)
			}
		}

	}

}

func (mongo *MongoCollection) Insert(docs ...interface{}) error {
	return mongo.collection.Insert(docs)
}

func (mongo *MongoCollection) FindId(id interface{}) (result []models.EventMessage, err error) {

	queryResult := mongo.collection.FindId(id)
	err = queryResult.All(&result)

	return result, err
}

func (mongo *MongoCollection) Find(query interface{}) (result []models.EventMessage, err error) {

	queryResult := mongo.collection.Find(query)
	err = queryResult.All(&result)

	return result, err
}

func (mongo *MongoCollection) RemoveId(id interface{}) error {
	return mongo.collection.RemoveId(id)
}

func (mongo *MongoCollection) Remove(selector interface{}) error {
	return mongo.collection.Remove(selector)
}
