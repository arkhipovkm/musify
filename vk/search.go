package vk

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"golang.org/x/net/html"
)

func loadHTML(query string, offset int, u *User) io.Reader {
	ch := make(chan []byte)
	go alSearch(query, offset, u, ch)
	body := <-ch
	// ioutil.WriteFile("al_search_body.html", body, os.ModePerm)
	bodyReader := bytes.NewReader(body)
	return bodyReader
}

func SearchQuery(query string, offset int, u *User) ([]Audio, error) {
	var err error
	// bodyReader, err := loadHTML(query, offset, u)
	bodyReader, err := os.Open("al_search_body.html")
	if err != nil {
		return nil, err
	}
	defer bodyReader.Close()
	z := html.NewTokenizer(bodyReader)
	var audios []Audio
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			acquireURLs(audios, u)
			for i := range audios {
				audios[i].DecypherURL(u)
			}
			return audios, err
		case tt == html.StartTagToken:
			t := z.Token()
			isAnchor := t.Data == "div"
			if isAnchor {
				for _, a := range t.Attr {
					if a.Key == "data-audio" {
						var rawAudio []interface{}
						err := json.Unmarshal([]byte(a.Val), &rawAudio)
						if err != nil {
							return nil, err
						}
						audio := NewAudio(rawAudio)
						audios = append(audios, audio)
						break
					}
				}
			}
		}
	}
}
