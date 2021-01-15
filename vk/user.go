package vk

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync/atomic"

	"golang.org/x/net/publicsuffix"
)

var VKAuthCounter uint64

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
		return
	}
	defer resp.Body.Close()
	bs, _ := ioutil.ReadAll(resp.Body)
	ss := string(bs)
	log.Println(ss)

	re = regexp.MustCompile("onLoginReCaptcha\\('(.*?)'")
	if re.MatchString(ss) {
		subm := re.FindStringSubmatch(ss)
		captchaSID := subm[1]
		log.Println("Captcha Needed. Captcha SID:", captchaSID)
		resp, err := http.Get("https://api.vk.com/captcha.php?sid=" + captchaSID)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		cbs, _ := ioutil.ReadAll(resp.Body)
		log.Println(cbs)
		log.Println(base64.URLEncoding.EncodeToString(cbs)
	}
	for _, cookie := range jar.Cookies(u) {
		if cookie.Name == "remixsid" {
			remixsid = cookie.Value
		} else if cookie.Name == "l" {
			userID, err = strconv.Atoi(cookie.Value)
			if err != nil {
				return
			}
		}
	}
	if remixsid == "" || userID == 0 {
		err = errors.New("Auth failed. Check credentials! ")
		return
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

// Authenticate performs the login procedure on VK using User's username and password.
// Adds User's RemixSID and ID to the user
func (u *User) Authenticate() error {
	var err error
	u.RemixSID, u.ID, err = login(u.login, u.password)
	atomic.AddUint64(&VKAuthCounter, 1)
	return err
}

// NewDefaultUser creates a new User with credentials retrieved from environmental variables
func NewDefaultUser() *User {
	var user User
	user.login = os.Getenv("VK_USERNAME")
	user.password = os.Getenv("VK_PASSWORD")
	return &user
}

// NewUser creates a new User with using credentials from arguments
func NewUser(login, password string) *User {
	return &User{
		login:    login,
		password: password,
	}
}
