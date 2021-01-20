package lyrics

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var TEMPLATE string = `
<html>
    <head>
        <meta charset="UTF-8">
    </head>
    <body>
        <h1>{{Title}}</h1>
        <h2>{{Subtitle}}</h2>
        <div class="cover">
            <img src="{{Cover}}">
        </div>
        <div class="content">
            {{Content}}
        </body>
    </body>
</html>`

func decodeBase64URI(base64EncodedURI string) (string, error) {
	decodedURI, err := base64.URLEncoding.DecodeString(base64EncodedURI)
	if err != nil {
		return "", err
	}
	return string(decodedURI), nil
}

func handleError(w *http.ResponseWriter, err error) {
	log.Println(err.Error())
	(*w).WriteHeader(http.StatusInternalServerError)
	(*w).Write([]byte(err.Error()))
}

func lyricsTemplate(artist, track, lyrics, coverURL string) []byte {
	body := []byte(TEMPLATE)

	paragraphs := strings.Split(lyrics, "\n")
	var content string = "<p>"
	for _, par := range paragraphs {
		if par != "" {
			content += par
			content += "<br>"
		} else {
			content += "</p>\n<p>"
		}
	}

	reTitle := regexp.MustCompile("\\{\\{Title\\}}")
	reSubtitle := regexp.MustCompile("\\{\\{Subtitle\\}\\}")
	reContent := regexp.MustCompile("\\{\\{Content\\}\\}")
	reCover := regexp.MustCompile("\\{\\{Cover\\}\\}")

	title := []byte(fmt.Sprintf(track))
	body = reTitle.ReplaceAll(body, title)

	subtitle := []byte(artist)
	body = reSubtitle.ReplaceAll(body, subtitle)

	bcontent := []byte(content)
	body = reContent.ReplaceAll(body, bcontent)

	body = reCover.ReplaceAll(body, []byte(coverURL))
	return body
}

func happiDevHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		r := recover()
		if r != nil {
			err, _ := r.(error)
			handleError(&w, err)
		}
	}()
	path := strings.Split(r.URL.Path, "/hlyrics/")[1]
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		err = errors.New("Ivalid URL format. Should be: /lyrics/:id_artist/:id_album/:id_track")
		handleError(&w, err)
		return
	}
	idArtist, err := strconv.Atoi(parts[0])
	if err != nil {
		handleError(&w, err)
		return
	}
	idAlbum, err := strconv.Atoi(parts[1])
	if err != nil {
		handleError(&w, err)
		return
	}
	idTrack, err := strconv.Atoi(parts[2])
	if err != nil {
		handleError(&w, err)
		return
	}

	lyrics, err := HappiGetLyrics(idArtist, idAlbum, idTrack)
	if err != nil {
		handleError(&w, err)
		return
	}

	// body, err := ioutil.ReadFile("template.html")
	// if err != nil {
	// 	handleError(&w, err)
	// 	return
	// }

	coverURL := "https://api.happi.dev/v1/music/cover/" + parts[1]
	body := lyricsTemplate(lyrics.Artist, lyrics.Track, lyrics.Lyrics, coverURL)

	w.Header().Add("Content-Type", "text/html")
	w.Write(body)
}

func auddDirectHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		r := recover()
		if r != nil {
			err, _ := r.(error)
			handleError(&w, err)
		}
	}()
	path := strings.Split(r.URL.Path, "/alyrics/")[1]
	parts := strings.Split(path, "/")

	base64EncodedLyrics := parts[0]
	lyrics, err := decodeBase64URI(base64EncodedLyrics)
	if err != nil {
		handleError(&w, err)
		return
	}

	base64EncodedCoverURL := parts[1]
	var coverURL string
	if base64EncodedCoverURL != "_" {
		coverURL, err = decodeBase64URI(base64EncodedCoverURL)
		if err != nil {
			handleError(&w, err)
			return
		}
	} else {
		coverURL = ""
	}

	base64EncodedArtist := parts[2]
	artist, err := decodeBase64URI(base64EncodedArtist)
	if err != nil {
		handleError(&w, err)
		return
	}

	base64EncodedTrack := parts[3]
	track, err := decodeBase64URI(base64EncodedTrack)
	if err != nil {
		handleError(&w, err)
		return
	}

	body := lyricsTemplate(artist, track, lyrics, coverURL)

	w.Header().Add("Content-Type", "text/html")
	w.Write(body)
}

func HappiDevLyricsServer() {
	http.HandleFunc("/hlyrics/", happiDevHandler)
}

func AuddDirectLyricsServer() {
	http.HandleFunc("/alyrics/", auddDirectHandler)
}
