package happidev

import (
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
        {{Content}}
    </body>
</html>`

func handleError(w *http.ResponseWriter, err error) {
	log.Println(err.Error())
	(*w).WriteHeader(http.StatusInternalServerError)
	(*w).Write([]byte(err.Error()))
}

func handler(w http.ResponseWriter, r *http.Request) {
	var err error
	path := strings.Split(r.URL.Path, "/lyrics/")[1]
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

	lyrics, err := GetLyrics(idArtist, idAlbum, idTrack)
	if err != nil {
		handleError(&w, err)
		return
	}

	// body, err := ioutil.ReadFile("template.html")
	// if err != nil {
	// 	handleError(&w, err)
	// 	return
	// }
	body := []byte(TEMPLATE)

	paragraphs := strings.Split(lyrics.Lyrics, "\n")
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

	title := []byte(fmt.Sprintf(lyrics.Track))
	body = reTitle.ReplaceAll(body, title)

	subtitle := []byte(lyrics.Artist)
	body = reSubtitle.ReplaceAll(body, subtitle)

	bcontent := []byte(content)
	body = reContent.ReplaceAll(body, bcontent)

	w.Header().Add("Content-Type", "text/html")
	w.Write(body)
}

func LyricsServer() {
	http.HandleFunc("/lyrics/", handler)
}
