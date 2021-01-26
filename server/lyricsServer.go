package server

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/arkhipovkm/musify/db"
	"github.com/arkhipovkm/musify/happidev"
)

var TEMPLATE string = `
<html>
    <head>
		<meta charset="UTF-8"/>
		<meta name="description" content="{{Description}}"/>
		<meta property="og:description" content="{{Description}}" />  
    </head>
	<body style="text-align: center; font-family: Century">
		<div class="cover" style="width: 100vw; height: 50vh; margin: auto">
			{{Cover}}
		</div>	
		<h1>{{Title}}</h1>
        <h2>{{Subtitle}}</h2>
        <div class="content">
			{{Content}}
		</div>
    </body>
</html>`

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
	if coverURL != "" {
		coverURL = fmt.Sprintf("<img src=\"%s\" style=\"opacity: 0.5; object-fit: contain; width:100%%; height:100%%;\"/>", coverURL)
	}

	reDescription := regexp.MustCompile("\\{\\{Description\\}\\}")
	reTitle := regexp.MustCompile("\\{\\{Title\\}\\}")
	reSubtitle := regexp.MustCompile("\\{\\{Subtitle\\}\\}")
	reContent := regexp.MustCompile("\\{\\{Content\\}\\}")
	reCover := regexp.MustCompile("\\{\\{Cover\\}\\}")

	title := []byte(track)
	body = reTitle.ReplaceAll(body, title)

	subtitle := []byte(artist)
	body = reSubtitle.ReplaceAll(body, subtitle)

	bcontent := []byte(content)
	body = reContent.ReplaceAll(body, bcontent)

	description := fmt.Sprintf("%s — %s Lyrics", artist, track)
	body = reDescription.ReplaceAll(body, []byte(description))

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

	lyrics, err := happidev.GetLyrics(idArtist, idAlbum, idTrack)
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

func idHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		r := recover()
		if r != nil {
			err, _ := r.(error)
			handleError(&w, err)
		}
	}()
	strID := strings.Split(r.URL.Path, "/ilyrics/")[1]
	intID, err := strconv.Atoi(strID)
	if err != nil {
		handleError(&w, err)
		return
	}
	lyrics, err := db.GetLyricsByID(intID)
	if err != nil {
		handleError(&w, err)
		return
	}
	if lyrics == nil {
		err = errors.New("Not Found")
		handleError(&w, err)
		return
	}
	body := lyricsTemplate(lyrics.Performer, lyrics.Title, lyrics.Text, lyrics.CoverURL)
	w.Header().Add("Content-Type", "text/html")
	w.Write(body)
}

func ServeHappiDevLyrics() {
	http.HandleFunc("/hlyrics/", happiDevHandler)
}

func ServeAuddDirectLyrics() {
	http.HandleFunc("/alyrics/", auddDirectHandler)
}

func ServeIDLyrics() {
	http.HandleFunc("/ilyrics/", idHandler)
}
