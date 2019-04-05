package persistient

type EventMessage struct {
	Channel    string `json:"channel"` //'event-updated' for example
	EventID    int64
	SenderID   int
	ReceiverID int
	Body       string `json:"body"`
}
