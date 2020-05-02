package download

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/arkhipovkm/musify/utils"
	"github.com/grafov/m3u8"
)

func httpFetch(uri string) ([]byte, error) {
	var err error
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 400 {
		return nil, errors.New(resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func fetchM3U8Playlist(uri string) (*m3u8.MediaPlaylist, string, error) {
	var err error
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(uri)
	if err != nil {
		return nil, uri, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 302 {
		redirectURL, err := resp.Location()
		if err != nil {
			return nil, uri, err
		}
		log.Printf("Redirect on fetchM3U8Playlist: %s\n", resp.Status)
		return fetchM3U8Playlist(redirectURL.String())
	}
	p, _, err := m3u8.DecodeFrom(resp.Body, false)
	if err != nil {
		return nil, uri, err
	}
	playlist := p.(*m3u8.MediaPlaylist)
	return playlist, uri, err
}

func fetchM3U8Segment(key, iv []byte, uri, path string, errChan chan error) {
	var err error
	data, err := httpFetch(uri)
	if err != nil {
		errChan <- err
	}
	if key != nil {
		block, err := aes.NewCipher(key)
		if err != nil {
			errChan <- err
		}
		if len(data)%aes.BlockSize != 0 {
			err = fmt.Errorf("Ciphertext is not a multiple of the block size. len(data)=%d, len(block)=%d", len(data), aes.BlockSize)
			errChan <- err
		}
		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(data, data)
	}
	err = ioutil.WriteFile(path, data, os.ModePerm)
	if err != nil {
		errChan <- err
	}
	errChan <- err
}

func fetchM3U8Track(uri, path string) error {
	var err error
	// parsedURI, _ := url.Parse(uri)
	myListPath := filepath.Join(path, "tslist.txt")
	myListFile, err := os.Create(myListPath)
	if err != nil {
		return err
	}
	// defer myListFile.Close()

	keySet := make(map[string]bool)
	keyValues := make(map[string][]byte)
	mediapl, uri, err := fetchM3U8Playlist(uri)
	if err != nil {
		return err
	}
	errChan := make(chan error, len(mediapl.Segments))
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
			if err != nil {
				return err
			}
			tsPath := filepath.Join(path, fmt.Sprintf("%d.ts", i))
			absoluteSegmentURI := filepath.ToSlash(filepath.Join(filepath.Dir(uri), segment.URI))
			absoluteSegmentURI = strings.ReplaceAll(absoluteSegmentURI, ":/", "://")
			go fetchM3U8Segment(keyValue, iv, absoluteSegmentURI, tsPath, errChan)
			i++
		}
	}
	for j := 0; j < int(i); j++ {
		err = <-errChan
		if err != nil {
			return err
		}
	}
	return err
}

// HLS downloads an audio using HLS protocol. Returns audio contents []byte
func HLS(uri string) ([]byte, error) {
	var err error
	path := filepath.Base(filepath.Dir(uri)) + "_" + utils.RandSeq(4)
	_ = os.Mkdir(path, os.ModePerm)
	defer os.RemoveAll(path)
	err = fetchM3U8Track(uri, path)
	if err != nil {
		return nil, err
	}
	t0 := time.Now()
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
		err = errors.New("ffmpeg: " + err.Error())
		return nil, err
	}

	t1 := time.Now()
	log.Printf("FFmpeg concat demuxer completed in %.1f ms\n", float64(t1.UnixNano()-t0.UnixNano())/float64(1e6))

	return out, nil
}

// HLSFile downloads an audio file using HLS protocol and writes the result into file
func HLSFile(uri string, filename string) (string, int, error) {
	var err error
	var n int
	if filename == "" {
		filename = filepath.Base(filepath.Dir(uri)) + "_" + utils.RandSeq(4) + ".mp3"
	}
	data, err := HLS(uri)
	if err != nil {
		return filename, n, err
	}
	n = len(data)
	err = ioutil.WriteFile(filename, data, os.ModePerm)
	if err != nil {
		return filename, n, err
	}
	return filename, n, err
}
