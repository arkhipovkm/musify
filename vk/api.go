package vk

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

var ACCESS_TOKEN string = os.Getenv("VK_API_ACCESS_TOKEN")

type ApiUser struct {
	ID             int
	First_name     string
	Last_name      string
	Photo_100      string
	Photo_max_orig string
}

func UsersGet(joinedIDs string) ([]*ApiUser, error) {
	var err error
	var users []*ApiUser
	if ACCESS_TOKEN != "" {
		URI := "https://api.vk.com/method/users.get"
		query := url.Values{
			"user_ids":     {joinedIDs},
			"v":            {"5.107"},
			"fields":       {"photo_100,photo_max_orig"},
			"access_token": {ACCESS_TOKEN},
		}
		resp, err := http.Get(URI + "?" + query.Encode())
		if err != nil {
			return users, err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return users, err
		}
		var payload map[string][]*ApiUser
		err = json.Unmarshal(body, &payload)
		if err != nil {
			return users, err
		}
		users = payload["response"]
	}
	return users, err
}
