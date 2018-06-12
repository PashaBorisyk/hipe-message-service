package main

import (
	"log"
	"room"
	"sync"
)

func main() {

	var wg sync.WaitGroup
	log.Print("Starting application...")

	wg.Add(1)
	go configureServerConnections(&wg)

	log.Println("Application started")
	wg.Wait()
	log.Print("Program finished")

}

func configureServerConnections(group *sync.WaitGroup) {

	defer group.Done()
	connectionsRoom := room.NewRoomFromConfig(group)
	repairCode := make(chan bool)
	group.Add(1)
	go connectionsRoom.InitServerConnection(&repairCode)
	group.Add(1)
	go connectionsRoom.InitClientConnections()

	for <-repairCode {
		group.Add(1)
		go connectionsRoom.InitServerConnection(&repairCode)
	}

}
