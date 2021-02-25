package happidev

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

type SearchResponseResult struct {
	Track     string
	IDTrack   int `json:"id_track"`
	Artist    string
	HasLyrics bool
	IDArtist  int `json:"id_artist"`
	Album     string
	BPM       int
	IDAlbum   int `json:"id_album"`
	Cover     string
	APIArtist string `json:"api_artist"`
	APIAlbums string `json:"api_albums"`
	APIAlbum  string `json:"api_album"`
	APITracks string `json:"api_tracks"`
	APITrack  string `json:"api_track"`
	APILyrics string `json:"api_lyrics"`
}

type SearchResponse struct {
	Success bool
	Length  int
	Error   string
	Result  []*SearchResponseResult
}

type LyricsResponseResult struct {
	Artist          string
	IDArtist        int `json:"id_artist"`
	Track           string
	IDTrack         int `json:"id_track"`
	IDAlbum         int `json:"id_album"`
	Album           string
	Lyrics          string
	APIArtist       string `json:"api_artist"`
	APIAlbums       string `json:"api_albums"`
	APIAlbum        string `json:"api_album"`
	APITracks       string `json:"api_tracks"`
	APITrack        string `json:"api_track"`
	APILyrics       string `json:"api_lyrics"`
	CopyrightLabel  string `json:"copyright_label"`
	CopyrightNotice string `json:"copyright_notice"`
	CopyrightString string `json:"copyright_string"`
}

type LyricsResponse struct {
	Success bool
	Length  int
	Error   string
	Result  *LyricsResponseResult
}

var API_TOKEN string = os.Getenv("HAPPIDEV_API_TOKEN")
var API_ENDPOINT string = "https://api.happi.dev/v1/music"

func httpGET(uri string) (*http.Response, error) {
	var err error
	var resp *http.Response
	client := &http.Client{}
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return resp, err
	}
	req.Header.Add("x-happi-key", API_TOKEN)
	resp, err = client.Do(req)
	return resp, err
}

func Search(q string) ([]*SearchResponseResult, error) {
	var err error
	var result []*SearchResponseResult
	query := make(url.Values)
	query.Add("q", q)
	query.Add("type", "track")

	uri := API_ENDPOINT + "?" + query.Encode()
	resp, err := httpGET(uri)
	if err != nil {
		return result, err
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		err = errors.New(resp.Status)
		return result, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}
	var searchResponse *SearchResponse
	err = json.Unmarshal(body, &searchResponse)
	if err != nil {
		return result, err
	}
	if !searchResponse.Success {
		err = errors.New(resp.Status + "." + searchResponse.Error)
		return result, err
	}
	if searchResponse.Length == 0 {
		err = errors.New("Empty Response")
		return result, err
	}
	result = searchResponse.Result
	return result, err
}

func GetLyrics(idArtist, idAlbum, idTrack int) (*LyricsResponseResult, error) {
	var err error
	var lyrics *LyricsResponseResult
	uri := fmt.Sprintf("%s/artists/%d/albums/%d/tracks/%d/lyrics",
		API_ENDPOINT,
		idArtist,
		idAlbum,
		idTrack,
	)
	resp, err := httpGET(uri)
	if err != nil {
		return lyrics, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		err = errors.New(resp.Status)
		return lyrics, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return lyrics, err
	}
	var lyricsResponse *LyricsResponse
	err = json.Unmarshal(body, &lyricsResponse)
	if err != nil {
		return lyrics, err
	}
	if !lyricsResponse.Success {
		err = errors.New(resp.Status + "." + lyricsResponse.Error)
		return lyrics, err
	}
	lyrics = lyricsResponse.Result
	return lyrics, err
}

func FindBestMatch(performer, title string, results []*SearchResponseResult) (*SearchResponseResult, error) {
	var err error
	var result *SearchResponseResult
	if len(results) == 0 {
		err = errors.New("Empty Results")
		return result, err
	}
	rePerformer := regexp.MustCompile("(?i)" + performer)
	reTitle := regexp.MustCompile("(?i)" + title)
	for i, item := range results {
		if rePerformer.MatchString(item.Artist) &&
			reTitle.MatchString(item.Track) {
			log.Println("Found best match in HappiDev results at #", i)
			return item, err
		}
	}
	result = results[0]
	log.Printf("No total good found in the results for %s - %s. Returning first item: %s - %s\n", performer, title, result.Artist, result.Track)
	return result, err
}
