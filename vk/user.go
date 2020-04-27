package vk

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"

	"golang.org/x/net/publicsuffix"
)

//login performs a login procedure on vk.com using username and password.
//Returns remixsid cookie value and user_id
func login(username, password string) (remixsid string, userID int, err error) {
	u, _ := url.Parse("https://login.vk.com/?act=login")
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	client := &http.Client{
		Jar: jar,
	}
	preResp, err := client.Get("https://vk.com")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer preResp.Body.Close()
	body, _ := ioutil.ReadAll(preResp.Body)

	re := regexp.MustCompile(`&ip_h=(.*?)&lg_h=(.*?)&`)
	groups := re.FindSubmatch(body)
	ipH := string(groups[1])
	lgH := string(groups[2])

	data := url.Values{
		"act":         []string{"login"},
		"email":       []string{username},
		"pass":        []string{password},
		"ip_h":        []string{ipH},
		"lg_h":        []string{lgH},
		"captcha_sid": []string{""},
		"captcha_key": []string{""},
		"expire":      []string{""},
		"role":        []string{"al_frame"},
	}
	resp, err := client.PostForm(u.String(), data)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer resp.Body.Close()
	for _, cookie := range jar.Cookies(u) {
		if cookie.Name == "remixsid" {
			remixsid = cookie.Value
		}
		if cookie.Name == "l" {
			userID, err = strconv.Atoi(cookie.Value)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	return
}

// User represents a VK User
type User struct {
	ID       int
	RemixSID string
	login    string
	password string
}

func (u *User) Authenticate() {
	var err error
	u.RemixSID, u.ID, err = login(u.login, u.password)
	if err != nil {
		panic(err)
	}
}

func GetDefaultUser() *User {
	var user User
	user.login = os.Getenv("MUSIFY_USERNAME")
	user.password = os.Getenv("MUSIFY_PASSWORD")
	return &user
}

func GetUser(login, password string) *User {
	return &User{
		login:    login,
		password: password,
	}
}
