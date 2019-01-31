package models

type MessageChannel chan EventMessage
type EventMessagesChannel map[int64]*MessageChannel

type EventMessage struct {
	Time                   int64       `json:"time"`
	EntityType             string      `json:"entityType"`
	EventMessageActionType string      `json:"eventMessageActionType"`
	EventID                int64       `json:"eventId"`
	UserID                 int64       `json:"userId"`
	Body                   interface{} `json:"body"`
}

type Message struct {
	Id       int64  `json:"id"`
	SenderId int64  `json:"senderId"`
	EventID  int    `json:"eventId"`
	Mills    int64  `json:"mills"`
	Message  string `json:"message"`
}

type Event struct {
	ID                int64   `json:"id"`
	CreatorID         int64   `json:"creatorId"`
	DateMills         int64   `json:"dateMills"`
	CreationDateMills int64   `json:"creationDateMills"`
	MaxMembers        int64   `json:"maxMembers"`
	Longitude         float64 `json:"longitude"`
	Latitude          float64 `json:"latitude"`
	CreatorNickname   string  `json:"creatorNickname"`
	Country           string  `json:"country"`
	City              string  `json:"city"`
	Street            string  `json:"street"`
	LocalName         string  `json:"localName"`
	Description       string  `json:"description"`
	IsPublic          bool    `json:"isPublic"`
	IsForOneGender    bool    `json:"isForOneGender"`
	IsForMale         bool    `json:"isForMale"`
	EventImageID      int64   `json:"eventImageId"`
	CreatorsImageURL  string  `json:"creatorsImageUrl"`
}
