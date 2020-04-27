package download

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grafov/m3u8"
)

func httpFetch(uri string) ([]byte, error) {
	var err error
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func fetchM3U8Playlist(url string) (*m3u8.MediaPlaylist, error) {
	var err error
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 302 {
		redirectURL, err := resp.Location()
		if err != nil {
			return nil, err
		}
		return fetchM3U8Playlist(redirectURL.String())
	}
	defer resp.Body.Close()
	p, _, err := m3u8.DecodeFrom(resp.Body, false)
	playlist := p.(*m3u8.MediaPlaylist)
	if err != nil {
		return nil, err
	}
	return playlist, err
}

func fetchM3U8Segment(key, iv []byte, uri, path string, wg *sync.WaitGroup) error {
	var err error
	defer wg.Done()
	data, err := httpFetch(uri)
	if err != nil {
		return err
	}
	if key != nil {
		block, err := aes.NewCipher(key)
		if err != nil {
			return err
		}
		if len(data)%aes.BlockSize != 0 {
			return fmt.Errorf("Ciphertext is not a multiple of the block size. len(data)=%d, len(block)=%d", len(data), aes.BlockSize)
		}
		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(data, data)
	}
	err = ioutil.WriteFile(path, data, os.ModePerm)
	if err != nil {
		return err
	}
	return err
}

func fetchM3U8Track(uri, path string) error {
	var err error
	// parsedURI, _ := url.Parse(uri)
	myListPath := filepath.Join(path, "tslist.txt")
	myListFile, err := os.Create(myListPath)
	if err != nil {
		return err
	}
	defer myListFile.Close()

	keySet := make(map[string]bool)
	keyValues := make(map[string][]byte)
	mediapl, err := fetchM3U8Playlist(uri)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var i uint8
	for _, segment := range mediapl.Segments {
		if segment != nil {
			var keyValue []byte
			var iv []byte
			if segment.Key != nil && segment.Key.Method != "NONE" {
				if keySet[segment.Key.URI] {
					keyValue = keyValues[segment.Key.URI]
				} else {
					keyValue, err = httpFetch(segment.Key.URI)
					if err != nil {
						return err
					}
					keySet[segment.Key.URI] = true
					keyValues[segment.Key.URI] = keyValue
				}
				iv = make([]byte, len(keyValue))
				iv[len(keyValue)-1] = i
			}
			_, err = myListFile.WriteString(fmt.Sprintf("file '%d.ts'\n", i))
			tsPath := filepath.Join(path, fmt.Sprintf("%d.ts", i))
			absoluteSegmentURI := filepath.ToSlash(filepath.Join(filepath.Dir(uri), segment.URI))
			absoluteSegmentURI = strings.ReplaceAll(absoluteSegmentURI, ":/", "://")
			go fetchM3U8Segment(keyValue, iv, absoluteSegmentURI, tsPath, &wg)
			wg.Add(1)
			if err != nil {
				return err
			}
			i++
		}
	}
	wg.Wait()
	return err
}

// HLS downloads an audio using HLS protocol. Returns audio contents []byte
func HLS(uri string) ([]byte, error) {
	var err error
	path := filepath.Base(filepath.Dir(uri))
	_ = os.Mkdir(path, os.ModePerm)
	defer os.RemoveAll(path)
	err = fetchM3U8Track(uri, path)
	if err != nil {
		return nil, err
	}
	out, err := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel",
		"panic",
		"-y",
		"-f",
		"concat",
		"-i",
		path+"/tslist.txt",
		"-c",
		"copy",
		"-map",
		"0:a:0",
		"-f",
		"mp3",
		"-",
	).Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HLSFile downloads an audio file using HLS protocol and writes the result into file
func HLSFile(uri string, filename string) (string, error) {
	var err error
	if filename == "" {
		filename = filepath.Base(filepath.Dir(uri)) + ".mp3"
	}
	data, err := HLS(uri)
	if err != nil {
		return filename, err
	}
	err = ioutil.WriteFile(filename, data, os.ModePerm)
	if err != nil {
		return filename, err
	}
	return filename, err
}
