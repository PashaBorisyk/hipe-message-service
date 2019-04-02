package rest

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

const (
	GetEventByMemberIdPath = "/event/get_by_member_id/"
)

func GetEventByMemberId(baseURL string,userId int) (eventIds []int64,err error) {
	url := baseURL + GetEventByMemberIdPath + strconv.FormatInt(int64(userId), 10)
	body,status,err := performGet(url)
	if status == 200 {
		err = json.Unmarshal(body,&eventIds)
		return eventIds,err
	}
	return nil,err
}

func performGet(url string) (body []byte,status int,err error)  {

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Unable to to perform GET request for URL : "+url, " ", err)
		return nil, 0, err
	}
	defer resp.Body.Close()

	status = resp.StatusCode
	if status == 200 || status == 204 {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Unable to read body from response", err)
			return nil,0,err
		}
		return body,status,nil
	}
	return nil,status,nil

}
