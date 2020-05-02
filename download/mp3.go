package download

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/arkhipovkm/musify/utils"
)

// MP3 downloads an audio and returns its contents []byte
func MP3(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

// MP3File downloads an audio and writes it to a file
func MP3File(uri, filename string) (string, int, error) {
	var err error
	var n int
	if filename == "" {
		filename = filepath.Base(filepath.Dir(uri)) + "_" + utils.RandSeq(4) + ".mp3"
	}
	resp, err := http.Get(uri)
	if err != nil {
		return filename, n, err
	}
	defer resp.Body.Close()

	file, err := os.Create(filename)
	if err != nil {
		return filename, n, err
	}
	defer file.Close()

	n, err = io.Copy(file, resp.Body)
	return filename, n, err
}
