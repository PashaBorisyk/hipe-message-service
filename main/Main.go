package main

import (
	"log"
	"partyfy-message-service/room"
	"sync"
)

func main() {

	var wg sync.WaitGroup
	log.Print("Starting application...")

	wg.Add(1)
	go start(&wg)

	log.Println("Application started")
	wg.Wait()
	log.Print("Program finished")

}

func start(group *sync.WaitGroup) {

	defer group.Done()
	connectionsRoom := room.NewRoomFromConfig(group)
	group.Add(1)
	go connectionsRoom.InitKafkaConnection()
	group.Add(1)
	go connectionsRoom.InitClientConnectionsHandler()

}
