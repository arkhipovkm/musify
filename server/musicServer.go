package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/arkhipovkm/id3-go"
	"github.com/arkhipovkm/musify/download"
	"github.com/arkhipovkm/musify/utils"
)

func httpGET(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func httpGETChan(uri string, dataChan chan []byte, errChan chan error) {
	data, err := httpGET(uri)
	dataChan <- data
	errChan <- err
}

func musicHandler(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	defer func() {
		r := recover()
		if r != nil {
			err, _ := r.(error)
			handleError(&w, err)
		}
	}()
	var err error
	path := strings.Split(r.URL.Path, "/streamer/")[1]
	base64EncodedURI := filepath.Base(filepath.Dir(path))
	decodedURI, err := decodeBase64URI(base64EncodedURI)
	if err != nil {
		handleError(&w, err)
		return
	}
	query := r.URL.Query()
	performer := query.Get("performer")
	title := query.Get("title")
	album := query.Get("album")
	year := query.Get("year")
	trck := query.Get("trck")
	base64EncodedApicCoverURI := query.Get("apic_cover")
	base64EncodedApicIconURI := query.Get("apic_icon")

	log.Println("Audio request:", performer, title, album, year, trck)

	errChan := make(chan error, 2)
	dataChan := make(chan []byte, 2)
	var apicCoverData, apicIconData []byte
	if base64EncodedApicCoverURI != "" {
		apicCoverURI, err := decodeBase64URI(base64EncodedApicCoverURI)
		if err != nil {
			handleError(&w, err)
			return
		}
		go httpGETChan(apicCoverURI, dataChan, errChan)
	} else {
		errChan <- nil
		dataChan <- nil
	}
	if base64EncodedApicIconURI != "" {
		apicIconURI, err := decodeBase64URI(base64EncodedApicIconURI)
		if err != nil {
			handleError(&w, err)
			return
		}
		go httpGETChan(apicIconURI, dataChan, errChan)
	} else {
		errChan <- nil
		dataChan <- nil
	}

	t1 := time.Now()
	log.Printf("Request prepared in: %.1f ms\n", float64(t1.UnixNano()-t0.UnixNano())/float64(1e6))

	var audioData []byte
	if strings.Contains(decodedURI, ".m3u8") {
		re := regexp.MustCompile("/[0-9a-f]+(/audios)?/([0-9a-f]+)/index.m3u8")
		replacedDecodedURI := re.ReplaceAllString(decodedURI, "$1/$2.mp3")
		if replacedDecodedURI != decodedURI {
			log.Printf("Downloading audio in MP3 mode: %s\n", replacedDecodedURI)
			audioData, err = download.MP3(string(replacedDecodedURI))
		} else {
			log.Printf("Downloading audio in HLS mode: %s\n", replacedDecodedURI)
			audioData, err = download.HLS(string(decodedURI))
		}
	} else if strings.Contains(decodedURI, ".mp3") {
		log.Printf("Downloading audio in MP3 mode: %s\n", decodedURI)
		audioData, err = download.MP3(string(decodedURI))
	} else {
		err = fmt.Errorf("unsupported file type: %s", filepath.Base(filepath.Dir(decodedURI)))
		handleError(&w, err)
		return
	}
	if err != nil {
		handleError(&w, err)
		return
	}

	for i := 0; i < 2; i++ {
		err := <-errChan
		if err != nil {
			handleError(&w, err)
			return
		}
	}

	apic0 := <-dataChan
	apic1 := <-dataChan

	if len(apic0) < len(apic1) {
		apicCoverData = apic1
		apicIconData = apic0
	} else {
		apicCoverData = apic0
		apicIconData = apic1
	}

	t2 := time.Now()
	log.Printf(
		"Fetched audio: %d bytes in %.1f ms, %.1f MB/s\n",
		len(audioData), float64(t2.UnixNano()-t1.UnixNano())/float64(1e6),
		float64(len(audioData))/float64(1e6)/(float64(t2.UnixNano()-t1.UnixNano())/float64(1e9)))

	id3Buffer, err := id3.ParseBuffer(audioData)
	if err == nil {
		utils.SetID3Tag(id3Buffer, performer, title, album, year, trck)
		utils.SetID3TagAPICs(id3Buffer, apicCoverData, apicIconData)
		id3Buffer.Close()
		audioData = id3Buffer.GetData()
	} else {
		log.Println("Error parsing ID3 tag:", err)
	}

	t3 := time.Now()
	log.Printf("Completed ID3 in: %.1f ms\n", float64(t3.UnixNano()-t2.UnixNano())/float64(1e6))

	w.Header().Add("Content-Type", "audio/mpeg")
	w.Write(audioData)

	t4 := time.Now()
	log.Printf(
		"Wrote response: %d bytes in %.1f ms, %.1f MB/s\n",
		len(audioData),
		float64(t4.UnixNano()-t3.UnixNano())/float64(1e6),
		float64(len(audioData))/float64(1e6)/(float64(t4.UnixNano()-t3.UnixNano())/float64(1e9)))

	t5 := time.Now()
	log.Printf("Request fulfilled in %.1f ms\n", float64(t5.UnixNano()-t0.UnixNano())/float64(1e6))

	log.Println("OK")
}

func ServeMusic() {
	http.HandleFunc("/streamer/", musicHandler)
}
