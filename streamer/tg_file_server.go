package streamer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func fileHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	filePath := strings.Split(r.URL.Path, "/file/")[1]
	uri := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", os.Getenv("TELEGRAM_BOT_API_TOKEN"), filePath)
	resp, err := http.Get(uri)
	if err != nil {
		handleError(&w, err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handleError(&w, err)
	}
	w.Header().Add("Content-Type", resp.Header.Get("Content-Type"))
	w.Write(data)
}

func TgFileServer() {
	http.HandleFunc("/file/", fileHandler)
}
