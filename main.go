package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/arkhipovkm/id3-go"
	"github.com/arkhipovkm/musify/download"
	"github.com/arkhipovkm/musify/utils"
	"github.com/arkhipovkm/musify/vk"
)

func search() {
	u := vk.GetDefaultUser()
	u.RemixSID = "38f12610b333afba92e6c7e15b74e59f252a89bcb70280cbe6773f8c822e0"
	u.ID = 5567597
	// u.Authenticate()
	audios, err := vk.SearchQuery("Muse", 0, u)
	if err != nil {
		panic(err)
	}
	for i, audio := range audios {
		u := strings.Split(audio.URL, "?")[0]
		fmt.Println(i, audio.Title, audio.Performer, u)
	}
}

func section() {
	u := vk.GetDefaultUser()
	u.Authenticate()
	nsec := time.Now().UnixNano()
	offset := 0
	n := 20
	playlistMap, audios, err := vk.SectionQuery("швец", offset, n, u)
	if err != nil {
		panic(err)
	}
	log.Println("Sec: ", float64(time.Now().UnixNano()-nsec)/float64(10e9))
	for _, a := range audios {
		u := strings.Split(a.URL, "?")[0]
		album := playlistMap[a.Album]
		var trck string
		for i, aa := range album.List {
			if aa.AudioID == a.AudioID {
				trck = strconv.Itoa(i+1) + "/" + strconv.Itoa(album.TotalCount)
				break
			}
		}
		// fmt.Println(a.Performer, a.Title, album.Title, a.CoverURLs, a.CoverURLp, u)
		if u != "" {
			query := make(url.Values)
			query.Set("performer", a.Performer)
			query.Set("title", a.Title)
			query.Set("album", album.Title)
			query.Set("year", album.YearInfoStr)
			query.Set("trck", trck)
			query.Set("apic_icon", base64.URLEncoding.EncodeToString([]byte(a.CoverURLs)))
			var apicCover string
			if album.CoverURL != "" {
				apicCover = album.CoverURL
			} else {
				apicCover = a.CoverURLp
			}
			query.Set("apic_cover", base64.URLEncoding.EncodeToString([]byte(apicCover)))
			uri := "http://localhost/" +
				base64.URLEncoding.EncodeToString([]byte(a.URL)) +
				"/" +
				url.PathEscape(strings.ReplaceAll(a.Performer, "/", "|")+" — "+strings.ReplaceAll(a.Title, "/", "|")) +
				".mp3" +
				"?" +
				query.Encode()
			fmt.Println(uri, len(uri))
		}
	}
}

func generateStreamerURIs() {

}

func hlsTest() {
	_, err := download.HLSFile(
		"https://psv4.vkuseraudio.net/c815126/u48331226/d9c350418/audios/93171c8143d0/index.m3u8?extra=5A6TeVyVvE36-CHWsVoCUUcwuS4G9k5Q3wmxU_w1nhdvrfYP1d1z0h1FOtUuugHfPrGJ25O_B38tDE0pBevT6JcZE64eUVyU252Gtj0gEvFpufeZd2qVbmHWhNe3sPOXmiybd5lDqlArxCr0rWs",
		"",
	)
	if err != nil {
		panic(err)
	}
}

func dl() {
	filename := "./test.mp3"
	url := "https://cs6-8v4.vkuseraudio.net/p19/2d9887e6a63553.mp3"
	download.MP3File(url, filename)
}

func testID3() {
	file, err := id3.Open("output.mp3")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	apicCover, err := ioutil.ReadFile("qcaUgVl9zGk.jpg")
	if err != nil {
		panic(err)
	}
	apicIcon, err := ioutil.ReadFile("da2IwiPg3pA.jpg")
	if err != nil {
		panic(err)
	}
	utils.SetID3Tag(file, "Luna", "Jukebox", "WTF", "2020", "1/10")
	utils.SetID3TagAPICs(file, apicIcon, apicCover)
}

func testID32() {
	file, err := id3.Open("1ac365572fb573.mp3")
	if err != nil {
		panic(err)
	}

	utils.SetID3Tag(file, "FOO", "Название", "Альбом", "2020", "1/10")
	fmt.Println(file.Tagger.Bytes())
	err = file.Close()
	if err != nil {
		panic(err)
	}
}

func readID3() {
	file, err := id3.Open("1ac365572fb573.mp3")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	fmt.Println(file.Artist(), []byte(file.Artist()), len(file.Artist()))
	fmt.Println(file.Title(), []byte(file.Title()), len(file.Title()))
	fmt.Println(file.Year(), []byte(file.Year()), len(file.Year()))
	fmt.Println(file.Album(), []byte(file.Album()), len(file.Album()))
}

func al() {

	u := vk.GetDefaultUser()
	// u.DummyAuthenticate()
	u.Authenticate()

	playlist := vk.LoadPlaylist("-2000940023_3940023_f93a1b113bbb77ef65", u)
	// playlist := alaudio.LoadPlaylist("5567597_-1_", u)
	playlist.AcquireURLs(u)
	playlist.DecypherURLs(u)
	for i, audio := range playlist.List {
		u := strings.Split(audio.URL, "?")[0]
		fmt.Println(i, audio.Title, audio.Performer, u, audio.CoverURLp, audio.CoverURLs)
	}
}

func main() {
	section()
}
