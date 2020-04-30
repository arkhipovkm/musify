package vk

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/arkhipovkm/musify/utils"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var VKErrorCounter uint64
var VKRequestCounter uint64

func check(k, v interface{}) {
	log.Panic(k, v)
}

var audioItemIndex = map[string]int{
	"AUDIO_ITEM_INDEX_ID":           0,
	"AUDIO_ITEM_INDEX_OWNER_ID":     1,
	"AUDIO_ITEM_INDEX_URL":          2,
	"AUDIO_ITEM_INDEX_TITLE":        3,
	"AUDIO_ITEM_INDEX_PERFORMER":    4,
	"AUDIO_ITEM_INDEX_DURATION":     5,
	"AUDIO_ITEM_INDEX_ALBUM_ID":     6,
	"AUDIO_ITEM_INDEX_AUTHOR_LINK":  8,
	"AUDIO_ITEM_INDEX_LYRICS":       9,
	"AUDIO_ITEM_INDEX_FLAGS":        10,
	"AUDIO_ITEM_INDEX_CONTEXT":      11,
	"AUDIO_ITEM_INDEX_EXTRA":        12,
	"AUDIO_ITEM_INDEX_HASHES":       13,
	"AUDIO_ITEM_INDEX_COVER_URL":    14,
	"AUDIO_ITEM_INDEX_ADS":          15,
	"AUDIO_ITEM_INDEX_SUBTITLE":     16,
	"AUDIO_ITEM_INDEX_MAIN_ARTISTS": 17,
	"AUDIO_ITEM_INDEX_FEAT_ARTISTS": 18,
	"AUDIO_ITEM_INDEX_ALBUM":        19,
	"AUDIO_ITEM_INDEX_TRACK_CODE":   20,

	"AUDIO_ITEM_HAS_LYRICS_BIT":     1,
	"AUDIO_ITEM_CAN_ADD_BIT":        2,
	"AUDIO_ITEM_CLAIMED_BIT":        4,
	"AUDIO_ITEM_HQ_BIT":             16,
	"AUDIO_ITEM_LONG_PERFORMER_BIT": 32,
	"AUDIO_ITEM_UMA_BIT":            128,
	"AUDIO_ITEM_REPLACEABLE":        512,
	"AUDIO_ITEM_EXPLICIT_BIT":       1024,
}

func checkVKError(outerArray []interface{}) error {
	var err error
	switch vkErr := outerArray[0].(type) {
	case string:
		vkErrI, _ := strconv.Atoi(vkErr)
		if vkErrI > 0 {
			err = fmt.Errorf("VK Error : %d", vkErrI)
			atomic.AddUint64(&VKErrorCounter, 1)
		}
	case int:
		if vkErr > 0 {
			err = fmt.Errorf("VK Error: %d", vkErr)
			atomic.AddUint64(&VKErrorCounter, 1)
		}
	}
	atomic.AddUint64(&VKRequestCounter, 1)
	return err
}

func extractPayload(body []byte) (map[string]interface{}, error) {
	var err error
	var payload map[string]interface{}
	re := regexp.MustCompile("<!--(.*?)$")
	subm := re.FindSubmatch(body)
	if len(subm) == 0 {
		err = fmt.Errorf("VK Error (payload). Body: %s", string(body))
		return nil, err
	}
	err = json.Unmarshal(subm[1], &payload)
	return payload, err
}

func extractRawAudios(payload map[string]interface{}) ([][]interface{}, error) {
	var err error
	var rawAudios [][]interface{}
	outerArray, _ := payload["payload"].([]interface{})
	err = checkVKError(outerArray)
	if err != nil {
		err = errors.New(err.Error() + " (rawAudios) ")
		return nil, err
	}
	innerArray, _ := outerArray[1].([]interface{})
	rawRawAudios, _ := innerArray[0].([]interface{})
	for _, rawAudio := range rawRawAudios {
		audio, _ := rawAudio.([]interface{})
		rawAudios = append(rawAudios, audio)
	}
	return rawAudios, err
}

func extractRawPlaylist(payload map[string]interface{}) (map[string]interface{}, error) {
	var err error
	var rawPlaylist map[string]interface{}
	outerArray, _ := payload["payload"].([]interface{})
	err = checkVKError(outerArray)
	if err != nil {
		err = errors.New(err.Error() + " (rawPlaylist) ")
		return nil, err
	}
	innerArray, _ := outerArray[1].([]interface{})
	if len(innerArray) == 0 {
		err = fmt.Errorf("VK Error: loadAudio response with no playlist: %s", payload)
		atomic.AddUint64(&VKErrorCounter, 1)
		return rawPlaylist, err
	}
	rawPlaylist, _ = innerArray[0].(map[string]interface{})
	return rawPlaylist, err
}

func extractRawSection(payload map[string]interface{}) (string, map[string]interface{}, error) {
	var err error
	var rawSection map[string]interface{}
	var rawHTML string
	outerArray, _ := payload["payload"].([]interface{})
	err = checkVKError(outerArray)
	if err != nil {
		err = errors.New(err.Error() + " (rawSection) ")
		return "", nil, err
	}
	innerArray, _ := outerArray[1].([]interface{})
	rawHTML, _ = innerArray[0].(string)
	rawSection, _ = innerArray[1].(map[string]interface{})
	return rawHTML, rawSection, err
}

func extractRawHTML(payload map[string]interface{}) ([]byte, error) {
	var err error
	var rawHTML string
	outerArray, _ := payload["payload"].([]interface{})
	err = checkVKError(outerArray)
	if err != nil {
		err = errors.New(err.Error() + " (rawAudios) ")
		return nil, err
	}
	innerArray, _ := outerArray[1].([]interface{})
	rawHTML, _ = innerArray[1].(string)
	return []byte(rawHTML), err
}

func doPOSTRequest(uri string, data url.Values, u *User) []byte {
	_url, _ := url.Parse(uri)
	cookie := http.Cookie{
		Name:  "remixsid",
		Value: u.RemixSID,
	}
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	jar.SetCookies(_url, []*http.Cookie{&cookie})
	client := &http.Client{
		Jar: jar,
	}
	resp, _ := client.PostForm(_url.String(), data)
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	utf8Body, _, _ := transform.Bytes(charmap.Windows1251.NewDecoder(), body)
	return utf8Body
}

func loadSectionPOST(ownerID, playlistID, offset int, accessHash string, u *User) []byte {
	data := url.Values{
		"act":            {"load_section"},
		"owner_id":       {strconv.Itoa(ownerID)},
		"playlist_id":    {strconv.Itoa(playlistID)},
		"offset":         {strconv.Itoa(offset)},
		"type":           {"playlist"},
		"al":             {"1"},
		"access_hash":    {accessHash},
		"is_loading_all": {"0"},
	}
	uri := "https://vk.com/al_audio.php"
	return doPOSTRequest(uri, data, u)
}

func reloadAudioPOST(ids []string, u *User) []byte {
	sIds := strings.Join(ids, ",")
	data := url.Values{
		"act": []string{"reload_audio"},
		"al":  []string{"1"},
		"ids": []string{sIds},
	}
	uri := "https://vk.com/al_audio.php"
	return doPOSTRequest(uri, data, u)
}

func searchPOST(query string, offset int, u *User) []byte {
	data := url.Values{
		"al":           {"1"},
		"offset":       {strconv.Itoa(offset)},
		"c[performer]": {"1"},
		"c[q]":         {query},
		"c[section]":   {"audio"},
	}
	uri := "https://vk.com/al_search.php"
	return doPOSTRequest(uri, data, u)
}

func sectionPOST(query string, u *User) []byte {
	data := url.Values{
		"act":      {"section"},
		"al":       {"1"},
		"q":        {query},
		"owner_id": {strconv.Itoa(u.ID)},
		"section":  {"search"},
		"claim":    {"0"},
		"is_layer": {"0"},
	}
	uri := "https://vk.com/al_audio.php"
	return doPOSTRequest(uri, data, u)
}

func reloadAudio(ids []string, u *User, ch chan [][]interface{}) {
	body := reloadAudioPOST(ids, u)
	payload, err := extractPayload(body)
	rawAudios, err := extractRawAudios(payload)
	if err != nil {
		log.Println("Reload Audio Error. Ids:", ids)
		log.Println(err)
	}
	ch <- rawAudios
}

func alSearch(query string, offset int, u *User, ch chan []byte) {
	body := searchPOST(query, offset, u)
	payload, err := extractPayload(body)
	rawHTML, err := extractRawHTML(payload)
	if err != nil {
		log.Println(err)
	}
	ch <- rawHTML
}

func alSection(query string, u *User) ([]string, *Playlist, error) {
	body := sectionPOST(query, u)
	payload, err := extractPayload(body)
	if err != nil {
		return nil, nil, err
	}
	rawHTML, rawSection, err := extractRawSection(payload)
	if err != nil {
		return nil, nil, err
	}
	rawSectionPlaylists, _ := rawSection["playlists"].([]interface{})
	var playlists []map[string]interface{}
	for _, rawSectionPlaylist := range rawSectionPlaylists {
		rawPlaylist := rawSectionPlaylist.(map[string]interface{})
		playlists = append(playlists, rawPlaylist)
	}

	re := regexp.MustCompile("href=\".music.album.(.*?)\"")
	subms := re.FindAllStringSubmatch(rawHTML, -1)

	uniquePlaylistIDs := make(map[string]bool)
	for _, subm := range subms {
		uniquePlaylistIDs[subm[1]] = true
	}
	var playlistIDs []string
	for k := range uniquePlaylistIDs {
		playlistIDs = append(playlistIDs, k)
	}
	var lastPlaylist *Playlist
	if len(playlists) > 0 {
		lastPlaylist = NewPlaylist(playlists[len(playlists)-1])
	}
	return playlistIDs, lastPlaylist, nil
}

func acquireURLs(audioList []*Audio, u *User) error {
	var err error
	var audioIds []string
	_ = os.MkdirAll(filepath.Join("cache", u.RemixSID, "audios"), os.ModePerm)
	for _, audio := range audioList {
		if audio.URL == "" {
			filename := filepath.Join("cache", u.RemixSID, "audios", fmt.Sprintf("%d_%d", audio.OwnerID, audio.AudioID))
			err = utils.ReadCache(filename, audio)
			if err != nil {
				continue
			}
		}
	}
	for _, audio := range audioList {
		if audio.URL == "" && audio.ActionHash != "" && audio.URLHash != "" {
			reloadID := fmt.Sprintf("%d_%d_%s_%s", audio.OwnerID, audio.AudioID, audio.ActionHash, audio.URLHash)
			audioIds = append(audioIds, reloadID)
		}
	}
	var chunks [][]string
	chunkSize := 10
	for i := 0; i < len(audioIds); i += chunkSize {
		end := i + chunkSize
		if end > len(audioIds) {
			end = len(audioIds)
		}
		chunks = append(chunks, audioIds[i:end])
	}
	ch := make(chan [][]interface{})
	for _, chunk := range chunks {
		go reloadAudio(chunk, u, ch)
	}

	audioIndex := make(map[int]*Audio)
	for i := range audioList {
		audio := audioList[i]
		audioIndex[audio.AudioID] = audio
	}

	for range chunks {
		rawAudios := <-ch
		for _, rawAudio := range rawAudios {
			audioURL, _ := rawAudio[audioItemIndex["AUDIO_ITEM_INDEX_URL"]].(string)
			audioIDf, _ := rawAudio[audioItemIndex["AUDIO_ITEM_INDEX_ID"]].(float64)
			audioID := int(audioIDf)
			audioIndex[audioID].URL = audioURL
		}
	}
	for _, audio := range audioList {
		fullID := fmt.Sprintf("%d_%d", audio.OwnerID, audio.AudioID)
		filename := filepath.Join("cache", u.RemixSID, "audios", fullID)
		_ = utils.WriteCache(filename, audio)
	}
	return err
}

// Playlist represents VK's audio playlist
type Playlist struct {
	Type           string
	OwnerID        int
	ID             int
	IsOfficial     bool
	Title          string
	Subtitle       string
	Description    string
	RawDescription string
	AuthorLine     string
	AuthorHref     string
	AuthorName     string
	InfoLine1      string
	InfoLine2      string
	LastUpdated    int
	Listens        string
	CoverURL       string
	EditHash       string
	IsFollowed     bool
	FollowHash     string
	AccessHash     string
	AddClasses     string
	GridCovers     string
	IsBlocked      bool
	List           []*Audio
	HasMore        bool
	NextOffset     int
	TotalCount     int
	TotalCountHash string
	YearInfoStr    string
	GenreInfoStr   string
	NPlaysInfoStr  string
}

// DecypherURLs decyphers playlist.List's audio URLs inplace.
func (playlist *Playlist) DecypherURLs(u *User) {
	for i := range playlist.List {
		playlist.List[i].DecypherURL(u)
	}
}

// AcquireURLs acquires URLs of playlist.List's audios by making *reload_audio* requests to vk's al_audio.php
// Makes concurrent requests.
// Might not work on large Lists of > 400 audios due to VK's throttling policy.
func (playlist *Playlist) AcquireURLs(u *User) {
	acquireURLs(playlist.List, u)
}

// AcquireURLsWG is the same as AcquireURLs, but with WaitGroup.Done() call in the end
// Allows you to acquireURLs in parallel with other goroutines
func (playlist *Playlist) AcquireURLsWG(u *User, wg *sync.WaitGroup) {
	acquireURLs(playlist.List, u)
	wg.Done()
}

// AcquireURLsChan is the same as AcquireURLs, but pipes the *Playlists to channel
func (playlist *Playlist) AcquireURLsChan(u *User, ch chan Playlist) {
	acquireURLs(playlist.List, u)
	ch <- *playlist
}

// FullID returns the ID of playlist in a form <OwnerID>_<ID>_<AccessHash>
func (playlist *Playlist) FullID() string {
	return strconv.Itoa(playlist.OwnerID) + "_" + strconv.Itoa(playlist.ID) + "_" + playlist.AccessHash
}

// NewPlaylist constructs Playlist from a raw playlist retrieved from vk's al_audio JSON responses
func NewPlaylist(rawPlaylist map[string]interface{}) *Playlist {
	playlist := Playlist{}
	var ok bool
	for k, v := range rawPlaylist {
		switch k {
		case "type":
			playlist.Type, ok = v.(string)
			if !ok {
				check(k, v)
			}
		case "ownerId":
			V, ok := v.(float64)
			if !ok {
				check(k, v)
			}
			playlist.OwnerID = int(V)
		case "id":
			switch V := v.(type) {
			case float64:
				playlist.ID = int(V)
			case string:
				playlist.ID = 0
			default:
				check(k, v)
			}
		case "isOfficial":
			switch V := v.(type) {
			case float64:
				playlist.IsOfficial = V > 0
			case bool:
				playlist.IsOfficial = V
			default:
				check(k, v)
			}
		case "title":
			playlist.Title, ok = v.(string)
			if !ok {
				check(k, v)
			}
		case "subTitle":
			switch V := v.(type) {
			case string:
				playlist.Subtitle = V
			default:
				playlist.Subtitle = ""
			}
		case "description":
			playlist.Description, ok = v.(string)
			if !ok {
				check(k, v)
			}
		case "coverUrl":
			playlist.CoverURL, ok = v.(string)
			if !ok {
				check(k, v)
			}
		case "hasMore":
			switch V := v.(type) {
			case float64:
				playlist.HasMore = V > 0
			case bool:
				playlist.HasMore = V
			default:
				check(k, v)
			}
		case "nextOffset":
			switch V := v.(type) {
			case float64:
				playlist.NextOffset = int(V)
			case string:
				playlist.NextOffset = 0
			default:
				check(k, v)
			}
		case "totalCount":
			V, ok := v.(float64)
			if !ok {
				check(k, v)
			}
			playlist.TotalCount = int(V)
		case "accessHash":
			V, ok := v.(string)
			if !ok {
				check(k, v)
			}
			playlist.AccessHash = V
		case "lastUpdated":
			V, ok := v.(float64)
			if !ok {
				check(k, v)
			}
			playlist.LastUpdated = int(V)
		case "authorName":
			V, ok := v.(string)
			if !ok {
				check(k, v)
			}
			if len(V) >= 8 && V[:8] == "<a class" {
				re := regexp.MustCompile(">(.*?)</a>")
				subm := re.FindStringSubmatch(V)
				playlist.AuthorName = subm[1]

				re2 := regexp.MustCompile("href=\"(.*?)\"")
				subm2 := re2.FindStringSubmatch(V)
				playlist.AuthorHref = subm2[1]
			}
		case "authorHref":
			V, ok := v.(string)
			if !ok {
				check(k, v)
			}
			if strings.Contains(V, "<a class") {
				re := regexp.MustCompile("href=\"(.*?)\"")
				subm := re.FindStringSubmatch(V)
				V = subm[1]
			}
			playlist.AuthorHref = V
		case "authorLine":
			V, ok := v.(string)
			if !ok {
				check(k, v)
			}
			playlist.AuthorLine = V
			if strings.Contains(V, "<a class") {
				re := regexp.MustCompile("href=\"(.*?)\"")
				subm := re.FindStringSubmatch(V)
				playlist.AuthorHref = subm[1]
			}
		case "infoLine1":
			V, ok := v.(string)
			if !ok {
				check(k, v)
			}
			if strings.Contains(V, "<span class=\"dvd\"></span>") {
				re := regexp.MustCompile("(.*?)<span class=\"dvd\"></span>(.*?)$")
				subm := re.FindAllStringSubmatch(V, -1)
				playlist.GenreInfoStr = subm[0][1]
				playlist.YearInfoStr = subm[0][2]
			}
			playlist.InfoLine1 = V
		case "infoLine2":
			V, ok := v.(string)
			if !ok {
				check(k, v)
			}
			if strings.Contains(V, "<span class=\"dvd\"></span>") {
				re := regexp.MustCompile("(.*?)<span class=\"dvd\"></span>(.*?)$")
				subm := re.FindAllStringSubmatch(V, -1)
				playlist.NPlaysInfoStr = subm[0][1]
			}
			playlist.InfoLine2 = V
		}

	}

	playlist.Title = html.UnescapeString(playlist.Title)
	playlist.Subtitle = html.UnescapeString(playlist.Subtitle)
	playlist.Description = html.UnescapeString(playlist.Description)
	playlist.AuthorName = html.UnescapeString(playlist.AuthorName)
	playlist.AuthorLine = html.UnescapeString(playlist.AuthorLine)

	audioList, _ := rawPlaylist["list"].([]interface{})
	for _, rawAudio := range audioList {
		rawAudio, _ := rawAudio.([]interface{})
		playlist.List = append(playlist.List, NewAudio(rawAudio))
	}

	return &playlist
}

// Audio represents VK's audio object
type Audio struct {
	AudioID         int
	OwnerID         int
	FullID          string
	Title           string
	Subtitle        string
	Performer       string
	Duration        int
	Lyrics          int
	URL             string
	Context         string
	Extra           string
	AddHash         string
	EditHash        string
	ActionHash      string
	DeleteHash      string
	ReplaceHash     string
	URLHash         string
	CoverURLs       string
	CoverURLp       string
	CanEdit         bool
	CanDelete       bool
	CanAdd          bool
	IsLongPerformer bool
	IsClaimed       bool
	IsExplicit      bool
	IsUMA           bool
	IsReplaceable   bool
	// Ads             string
	Album     string
	AlbumID   int
	TrackCode string
}

// DecypherURL decyphers inplace Audio's URL using caesar cypher as in VK Audioplayer
func (audio *Audio) DecypherURL(u *User) error {
	var err error
	if audio.URL != "" && strings.Contains(audio.URL, "audio_api_unavailable") {
		audio.URL, err = decypherAudioURL(audio.URL, u.ID)
	}
	if err != nil {
		log.Panic(err)
	}
	return err
}

// NewAudio accepts a rawAudio []interface{} coming from Playlist.List and returns an Audio object
func NewAudio(rawAudio []interface{}) *Audio {
	audio := Audio{}
	var flags int

	var hashes string
	var Hashes []string

	var covers string
	var Covers []string
	var ok bool
	for i, v := range rawAudio {
		switch i {
		case audioItemIndex["AUDIO_ITEM_INDEX_ID"]:
			V, ok := v.(float64)
			if !ok {
				check(i, v)
			}
			audio.AudioID = int(V)
		case audioItemIndex["AUDIO_ITEM_INDEX_OWNER_ID"]:
			V, ok := v.(float64)
			if !ok {
				check(i, v)
			}
			audio.OwnerID = int(V)
		case audioItemIndex["AUDIO_ITEM_INDEX_TITLE"]:
			audio.Title, ok = v.(string)
			if !ok {
				check(i, v)
			}
		case audioItemIndex["AUDIO_ITEM_INDEX_SUBTITLE"]:
			audio.Subtitle, ok = v.(string)
			if !ok {
				check(i, v)
			}
		case audioItemIndex["AUDIO_ITEM_INDEX_PERFORMER"]:
			audio.Performer, ok = v.(string)
			if !ok {
				check(i, v)
			}
		case audioItemIndex["AUDIO_ITEM_INDEX_DURATION"]:
			V, ok := v.(float64)
			if !ok {
				check(i, v)
			}
			audio.Duration = int(V)
		case audioItemIndex["AUDIO_ITEM_INDEX_LYRICS"]:
			V, ok := v.(float64)
			if !ok {
				check(i, v)
			}
			audio.Lyrics = int(V)
		case audioItemIndex["AUDIO_ITEM_INDEX_URL"]:
			audio.URL, ok = v.(string)
			if !ok {
				check(i, v)
			}
		case audioItemIndex["AUDIO_ITEM_INDEX_FLAGS"]:
			flags, ok = v.(int)
		case audioItemIndex["AUDIO_ITEM_INDEX_CONTEXT"]:
			audio.Context, ok = v.(string)
			if !ok {
				check(i, v)
			}
		case audioItemIndex["AUDIO_ITEM_INDEX_EXTRA"]:
			audio.Extra, ok = v.(string)
			if !ok {
				check(i, v)
			}
		case audioItemIndex["AUDIO_ITEM_INDEX_ALBUM"]:
			switch V := v.(type) {
			case []interface{}:
				audio.Album = fmt.Sprintf("%.f_%.f_%s", V...)
			default:
				audio.Album = ""
			}
		case audioItemIndex["AUDIO_ITEM_INDEX_ALBUM_ID"]:
			V, ok := v.(float64)
			if !ok {
				check(i, V)
			}
			audio.AlbumID = int(V)
		case audioItemIndex["AUDIO_ITEM_INDEX_TRACK_CODE"]:
			audio.TrackCode, ok = v.(string)
			if !ok {
				check(i, v)
			}
		case audioItemIndex["AUDIO_ITEM_INDEX_HASHES"]:
			hashes, ok = v.(string)
			Hashes = strings.Split(hashes, "/")
		case audioItemIndex["AUDIO_ITEM_INDEX_COVER_URL"]:
			covers, ok = v.(string)
			Covers = strings.Split(covers, ",")
		}

	}
	audio.CoverURLs = Covers[0]
	if len(Covers) > 1 {
		audio.CoverURLp = Covers[1]
	} else {
		audio.CoverURLp = ""
	}

	audio.AddHash = Hashes[0]
	audio.EditHash = Hashes[1]
	audio.ActionHash = Hashes[2]
	audio.DeleteHash = Hashes[3]
	audio.ReplaceHash = Hashes[4]
	audio.URLHash = Hashes[5]

	audio.CanEdit = audio.EditHash != ""
	audio.CanDelete = audio.DeleteHash != ""
	audio.CanAdd = (flags & audioItemIndex["AUDIO_ITEM_CAN_ADD_BIT"]) != 0
	audio.IsLongPerformer = (flags & audioItemIndex["AUDIO_ITEM_LONG_PERFORMER_BIT"]) != 0
	audio.IsClaimed = (flags & audioItemIndex["AUDIO_ITEM_CLAIMED_BIT"]) != 0
	audio.IsExplicit = (flags & audioItemIndex["AUDIO_ITEM_EXPLICIT_BIT"]) != 0
	audio.IsUMA = (flags & audioItemIndex["AUDIO_ITEM_UMA_BIT"]) != 0
	audio.IsReplaceable = (flags & audioItemIndex["AUDIO_ITEM_REPLACEABLE"]) != 0

	audio.Title = html.UnescapeString(audio.Title)
	audio.Subtitle = html.UnescapeString(audio.Subtitle)
	audio.Performer = html.UnescapeString(audio.Performer)

	return &audio
}

//LoadPlaylist downloads and parses a Playlist by id.
func LoadPlaylist(id string, u *User) *Playlist {
	var err error
	playlist := new(Playlist)
	s := strings.Split(id, "_")
	ownerID, _ := strconv.Atoi(s[0])
	playlistID, _ := strconv.Atoi(s[1])
	var accessHash string
	if len(s) > 2 {
		accessHash = s[2]
	} else {
		log.Println("LoadPlaylist: short ID:", id)
	}
	_ = os.MkdirAll(filepath.Join("cache", u.RemixSID, "playlists"), os.ModePerm)
	filename := filepath.Join("cache", u.RemixSID, "playlists", fmt.Sprintf("%d_%d", ownerID, playlistID))
	err = utils.ReadCache(filename, playlist)
	if err == nil {
		return playlist
	}

	body := loadSectionPOST(ownerID, playlistID, 0, accessHash, u)
	payload, err := extractPayload(body)
	rawPlaylist, err := extractRawPlaylist(payload)
	if err != nil {
		log.Println("Load Playlist Error. Id:", id)
		log.Println(err)
		return playlist
	}
	playlist = NewPlaylist(rawPlaylist)
	_ = utils.WriteCache(filename, playlist)
	return playlist
}

//LoadPlaylistChan is the same as LoadPlaylist, but pipes Playlist to channel
func LoadPlaylistChan(id string, u *User, ch chan *Playlist) {
	pl := LoadPlaylist(id, u)
	ch <- pl
}

//LoadAudio downloads and parses an Audio by id
func LoadAudio(id string, u *User) *Audio {
	ch := make(chan [][]interface{})
	go reloadAudio([]string{id}, u, ch)
	rawAudios := <-ch
	audio := NewAudio(rawAudios[0])
	audio.DecypherURL(u)
	return audio
}

//LoadAudioChan is the same as LoadAudio, but pipes Audio to channel
func LoadAudioChan(id string, u *User, ch chan *Audio, wg *sync.WaitGroup) {
	a := LoadAudio(id, u)
	wg.Done()
	ch <- a
}
