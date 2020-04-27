package download

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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
func MP3File(uri, filename string) (string, error) {
	var err error
	if filename == "" {
		filename = filepath.Base(filepath.Dir(uri)) + ".mp3"
	}
	resp, err := http.Get(uri)
	if err != nil {
		return filename, err
	}
	defer resp.Body.Close()

	file, err := os.Create(filename)
	if err != nil {
		return filename, err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return filename, err
}
