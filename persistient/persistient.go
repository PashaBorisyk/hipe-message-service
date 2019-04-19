package persistient

type EventMessage struct {
	Channel    string      `json:"channel" bson:"channel"` //'event-updated' for example
	EventID    int64       `json:"eventID" bson:"eventID"`
	SenderID   int         `json:"senderID" bson:"senderID"`
	ReceiverID int         `json:"receiverID" bson:"receiverID"`
	IsSent     bool        `json:"-" bson:"isSent"`
	Body       interface{} `json:"body" `
	ID         interface{} `json:"_id" bson:"-"`
}
