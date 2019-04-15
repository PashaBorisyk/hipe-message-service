package persistient

type EventMessage struct {
	Channel    string `json:"channel"` //'event-updated' for example
	EventID    int64 `json:"-"`
	SenderID   int `json:"-"`
	ReceiverID int `json:"-"`
	Body       interface{} `json:"body"`
}
