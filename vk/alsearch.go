package vk

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
)

type Page struct {
	ID         int
	Name       string
	PhotoURL   string
	ScreenName string
	Type       string
	Info       string
	IsOfficial bool
}

func NewPage(idStr string, rawPage []interface{}) (*Page, error) {
	var err error
	idInt, err := strconv.Atoi(idStr[:len(idStr)-1])
	if err != nil {
		return nil, err
	}
	page := &Page{
		ID: idInt,
	}
	for i, v := range rawPage {
		switch i {
		case 0:
			V, ok := v.(string)
			if !ok {
				log.Panic("NewPage parse error. Item:", i)
			}
			page.Name = V
		case 1:
			V, ok := v.(string)
			if !ok {
				log.Panic("NewPage parse error. Item:", i)
			}
			page.PhotoURL = V
		case 2:
			V, ok := v.(string)
			if !ok {
				log.Panic("NewPage parse error. Item:", i)
			}
			page.ScreenName = V[1:]
		case 3:
			V, ok := v.(string)
			if !ok {
				log.Panic("NewPage parse error. Item:", i)
			}
			page.Type = V
		case 4:
			V, ok := v.(string)
			if !ok {
				log.Panic("NewPage parse error. Item:", i)
			}
			infos := strings.Split(V, ", ")
			page.Info = infos[0]
		case 5:
			V, ok := v.(float64)
			if !ok {
				log.Panic("NewPage parse error. Item:", i)
			}
			page.IsOfficial = int(V) != 0
		}
	}
	return page, err
}

func extractRawPages(payload map[string]interface{}) (map[string][]interface{}, error) {
	var err error
	rawPages := make(map[string][]interface{})
	outerArray, _ := payload["payload"].([]interface{})
	err = checkVKError(outerArray)
	if err != nil {
		err = errors.New(err.Error() + " (rawPages) ")
		return nil, err
	}
	innerArray, _ := outerArray[1].([]interface{})
	rawRawPages, _ := innerArray[0].(map[string]interface{})
	for k, rawPage := range rawRawPages {
		page, _ := rawPage.([]interface{})
		rawPages[k] = page
	}
	return rawPages, err
}

func pagesPOST(query string, u *User) []byte {
	data := url.Values{
		"act": {"get_pages_hints"},
		"al":  {"1"},
		"q":   {query},
	}
	uri := "https://vk.com/al_search.php"
	return doPOSTRequest(uri, data, u)
}

func LoadPages(query string, u *User) ([]*Page, error) {
	var err error
	var pages []*Page
	body := pagesPOST(query, u)
	payload, err := extractPayload(body)
	if err != nil {
		return nil, err
	}
	rawPages, err := extractRawPages(payload)
	if err != nil {
		return nil, err
	}
	for k, rawPage := range rawPages {
		page, err := NewPage(k, rawPage)
		if err != nil {
			log.Println("Skipping page:", k)
			continue
		}
		pages = append(pages, page)
	}
	return pages, err
}

func PagesQuery(query string, u *User) ([]*Page, map[int]*Playlist, error) {
	var err error
	var pages []*Page
	allPages, err := LoadPages(query, u)

	var playlistIDs []string
	for _, page := range allPages {
		if page.Type != "s_groups" {
			plID := fmt.Sprintf("%d_-1", page.ID)
			playlistIDs = append(playlistIDs, plID)
		}
	}

	ch := make(chan *Playlist, len(playlistIDs))
	for _, plID := range playlistIDs {
		go LoadPlaylistChan(plID, u, ch)
	}

	playlistMap := make(map[int]*Playlist)
	m := len(playlistIDs)
	for i := 0; i < m; i++ {
		pl := <-ch
		playlistMap[pl.OwnerID] = pl
	}

	var userIDs string
	for _, page := range allPages {
		pl := playlistMap[page.ID]
		if pl != nil && len(pl.List) > 0 {
			userIDs += strconv.Itoa(page.ID) + ","
			pages = append(pages, page)
		}
	}

	users, err := usersGet(userIDs)
	if err != nil {
		panic(err)
	}
	for _, user := range users {
		for _, page := range pages {
			if page.ID == user.ID {
				page.PhotoURL = user.Photo_100
			}
		}
		playlistMap[user.ID].CoverURL = user.Photo_max_orig
	}
	return pages, playlistMap, err
}
