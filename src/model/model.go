package model

import "math/rand"

type Event struct {
	Longitude       float64 `json:"longtitude"`
	Latitude        float64 `json:"latitude"`
	CreatorNickname string  `json:"creatorNickname"`
	Description     string  `json:"description"`
	CreatorImageUrl string  `json:"creatorImageUrl"`
}

func NewModel() *Event {
	return &Event{
		Longitude:       rand.Float64(),
		Latitude:        rand.Float64(),
		CreatorImageUrl: "https://pp.userapi.com/c837528/v837528158/2a440/-IxCca3vLc4.jpg?ava=1",
		Description:     "You can be my girl, i can be your man",
		CreatorNickname: "pashaborisyk",
	}
}
