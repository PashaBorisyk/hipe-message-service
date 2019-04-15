package rest

import (
	"config"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

const (
	getEventByMemberIdPath   = "/event/get_ids_by_member_id/"
	getUsersIdsByEventIdPath = "/user/get_users_ids_by_event_id/"
	loginPath                = "/user/login/?username=message-processor&secret=password"
)

var baseURL string
var token = ""

func init() {
	baseURL = config.GetConfig().ConnectionsConfig.EventProcessor.Url
	token, _ = login()
}

func login() (string, error) {

	if token != "" {
		return token,nil
	}

	log.Println("Trying to login to Main Event Processor")

	url := baseURL + loginPath
	body, status, err := performGet(url)
	if status != 200 || err != nil {
		log.Println("Failed to login with url : ", loginPath, "Will retry to login each request")
		return "", err
	}

	log.Println("Login successful. Token is: ", string(body[:10]),"....")
	return string(body), nil
}

func GetEventsIDsMemberID(userID int) (eventIds []int64, err error) {

	log.Println("Getting events for user with id : ", userID)

	if token,err = login(); err != nil {
		log.Println("Will not perform request without token")
		return nil, err
	}

	url := baseURL + getEventByMemberIdPath + strconv.FormatInt(int64(userID), 10) + "/"
	body, status, err := performGet(url)
	if status == 200 {
		err = json.Unmarshal(body, &eventIds)
		return eventIds, err
	}
	return nil, err
}

func GetUserIDsByEventID(eventId int64) (userIds []int, err error) {

	if token,err = login(); err != nil {
		log.Println("Will not perform request without token")
		return nil, err
	}

	url := baseURL + getUsersIdsByEventIdPath + strconv.FormatInt(eventId, 10) + "/"
	body, status, err := performGet(url)
	if status == 200 {
		err = json.Unmarshal(body, &userIds)
	}
	return userIds, err
}

func performGet(url string) (body []byte, status int, err error) {

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("AUTHORIZATION", token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Unable to to perform GET request for URL : ", url, " ", err)
		return nil, 0, err
	}
	defer resp.Body.Close()

	status = resp.StatusCode
	if status == 200 || status == 201 {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Unable to read body from response", err)
			return nil, 0, err
		}
		return body, status, nil
	}
	return nil, status, nil

}
