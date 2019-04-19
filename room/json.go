package room

import (
	"encoding/json"
	"log"
)

func unmarshalEventActionRecord(raw []byte) (record *EventActionRecord) {
	record = new(EventActionRecord)
	err := json.Unmarshal(raw, record)
	if err != nil {
		log.Println("Unable to unmarshal ", string(raw), "to EventUpdatedRecord")
		log.Println(err)
		return nil
	}
	return record
}

func unmarshalEventUserActionRecord(raw []byte) (record *EventUserActionRecord) {
	record = new(EventUserActionRecord)
	err := json.Unmarshal(raw, record)
	if err != nil {
		log.Println("Unable to unmarshal ", string(raw), "to EventUserActionRecord")
		log.Println(err)
		return nil
	}
	return record
}

func unmarshalUserActionRecord(raw []byte) (record *UserActionRecord) {
	record = new(UserActionRecord)
	err := json.Unmarshal(raw, record)
	if err != nil {
		log.Println("Unable to unmarshal ", string(raw), "to EventUserActionRecord")
		log.Println(err)
		return nil
	}
	return record
}
func unmarshalImageAddedRecord(raw []byte) (record *MultipleUserEventActionRecord) {
	record = new(MultipleUserEventActionRecord)
	err := json.Unmarshal(raw, record)
	if err != nil {
		log.Println("Unable to unmarshal ", string(raw), "to MultipleUserEventActionRecord")
		log.Println(err)
		return nil
	}
	return record
}
