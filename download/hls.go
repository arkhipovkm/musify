package download

import (
	"bytes"
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
		return nil, fmt.Errorf("%d: %s", resp.StatusCode, resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func FetchM3U8Playlist(uri string) (*m3u8.MediaPlaylist, string, error) {
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
		return FetchM3U8Playlist(redirectURL.String())
	}
	p, _, err := m3u8.DecodeFrom(resp.Body, false)
	if err != nil {
		return nil, uri, err
	}
	playlist := p.(*m3u8.MediaPlaylist)
	return playlist, uri, err
}

type fetchedSegmentResult struct {
	i    uint8
	data []byte
	err  error
}

func FetchM3U8Segment(i uint8, key []byte, uri string, resultChan chan fetchedSegmentResult) {
	var err error
	data, err := httpFetch(uri)
	if err != nil {
		resultChan <- fetchedSegmentResult{
			i:    i,
			data: data,
			err:  err,
		}
	}
	if key != nil {

		iv := make([]byte, len(key))
		iv[len(key)-1] = i

		block, err := aes.NewCipher(key)
		if err != nil {
			resultChan <- fetchedSegmentResult{
				i:    i,
				data: data,
				err:  err,
			}
		}
		if len(data)%aes.BlockSize != 0 {
			err = fmt.Errorf("ciphertext is not a multiple of the block size. len(data)=%d, len(block)=%d", len(data), aes.BlockSize)
			resultChan <- fetchedSegmentResult{
				i:    i,
				data: data,
				err:  err,
			}
		}
		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(data, data)
	}
	resultChan <- fetchedSegmentResult{
		i:    i,
		data: data,
		err:  err,
	}
}

func FetchM3U8Track(uri string) ([]byte, error) {
	var err error
	var tsData []byte

	keySet := make(map[string]bool)
	keyValues := make(map[string][]byte)
	mediapl, uri, err := FetchM3U8Playlist(uri)
	if err != nil {
		return tsData, err
	}

	segCnt := 0
	for _, seg := range mediapl.Segments {
		if seg != nil {
			segCnt++
		}
	}
	resultChan := make(chan fetchedSegmentResult, segCnt)

	var i uint8
	for _, segment := range mediapl.Segments {
		if segment != nil {
			var keyValue []byte
			if segment.Key != nil && segment.Key.Method != "NONE" {
				if keySet[segment.Key.URI] {
					keyValue = keyValues[segment.Key.URI]
				} else {
					keyValue, err = httpFetch(segment.Key.URI)
					if err != nil {
						return tsData, err
					}
					keySet[segment.Key.URI] = true
					keyValues[segment.Key.URI] = keyValue
				}
			}

			absoluteSegmentURI := filepath.ToSlash(filepath.Join(filepath.Dir(uri), segment.URI))
			absoluteSegmentURI = strings.ReplaceAll(absoluteSegmentURI, ":/", "://")
			go FetchM3U8Segment(i, keyValue, absoluteSegmentURI, resultChan)
			i++
		}
	}
	chunks := make([][]byte, segCnt)
	for jj := 0; jj < int(i); jj++ {
		res := <-resultChan
		if res.err != nil {
			return tsData, res.err
		}
		chunks[res.i] = res.data
		if jj == segCnt-1 {
			break
		}
	}
	for _, chunk := range chunks {
		tsData = append(tsData, chunk...)
	}
	return tsData, err
}

// HLS downloads an audio using HLS protocol. Returns audio contents []byte
func HLS(uri string) ([]byte, error) {
	var err error
	var out []byte

	t0 := time.Now()
	ts, err := FetchM3U8Track(uri)
	if err != nil {
		return nil, err
	}
	t1 := time.Now()
	log.Printf(
		"HLS MPEG-TS downloaded in %.1f ms, %.1fMiB/s\n",
		float64(t1.UnixNano()-t0.UnixNano())/float64(1e6),
		(float64(len(ts))/float64(1024*1024))/(float64(t1.UnixNano()-t0.UnixNano())/float64(1e9)),
	)

	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel",
		"panic",
		"-y",
		"-i",
		"-",
		"-c",
		"copy",
		"-map",
		"0:a:0",
		"-f",
		"mp3",
		"-",
	)
	cmd.Stdin = bytes.NewReader(ts)
	out, err = cmd.Output()

	if err != nil {
		err = errors.New("ffmpeg erorr: " + err.Error())
		return nil, err
	}

	t2 := time.Now()
	log.Printf(
		"FFmpeg concat demuxer completed in %.1f ms\n",
		float64(t2.UnixNano()-t1.UnixNano())/float64(1e6),
	)

	return out, err
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
