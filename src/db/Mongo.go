package db

import (
	"config"
	"gopkg.in/mgo.v2"
	"log"
)

var collection *mgo.Collection

func init() {

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

	database := session.DB(mongoConfig.Database)

	collectionsNames, err := database.CollectionNames()
	if err != nil {
		log.Print("Error occured while getting collections names")
		log.Print(err)
		return
	}

	for _, name := range collectionsNames {
		if name == mongoConfig.Collection {
			hasName = true
			break
		}
	}

	if !hasName {
		log.Print("No such collection found : " + mongoConfig.Collection)
		return
	}

	collection = database.C(mongoConfig.Collection)

}
