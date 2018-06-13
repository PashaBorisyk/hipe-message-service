package model

type MessageChannel chan EventMessage
type EventBasket map[int64]*MessageChannel

type EventMessage struct {
	Time     int64  `json:"time"`
	Type     int    `json:"type"`
	Category int    `json:"category"`
	Body     string `json:"body"`
	EventId  int64  `json:"eventId"`
}

type Message struct {
	Id          int64  `json:"id"`
	SenderId    int64  `json:"senderId"`
	EventID     int    `json:"eventId"`
	Mills       int64  `json:"mills"`
	Message     string `json:"message"`
	Informative bool   `json:"informative"`
	Execute     int    `json:"execute"`
}

