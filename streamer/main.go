package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/arkhipovkm/id3-go"
	"github.com/arkhipovkm/musify/download"
	"github.com/arkhipovkm/musify/utils"
)

func decodeBase64URI(base64EncodedURI string) (string, error) {
	decodedURI, err := base64.URLEncoding.DecodeString(base64EncodedURI)
	if err != nil {
		return "", err
	}
	return string(decodedURI), nil
}

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

func handleError(w *http.ResponseWriter, err error) {
	log.Println(err.Error())
	(*w).WriteHeader(http.StatusInternalServerError)
	(*w).Write([]byte(err.Error()))
}

func handler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		r := recover()
		err, _ := r.(error)
		handleError(&w, err)
	}()
	var err error
	base64EncodedURI := filepath.Base(filepath.Dir(r.URL.Path))
	log.Println(base64EncodedURI)
	decodedURI, err := decodeBase64URI(base64EncodedURI)
	if err != nil {
		handleError(&w, err)
		return
	}
	log.Println("Received an audio to stream: ", decodedURI)
	query := r.URL.Query()
	log.Println("Query:", query)
	performer := query.Get("performer")
	title := query.Get("title")
	album := query.Get("album")
	year := query.Get("year")
	trck := query.Get("trck")
	base64EncodedApicCoverURI := query.Get("apic_cover")
	base64EncodedApicIconURI := query.Get("apic_icon")

	errChan := make(chan error, 2)
	dataChan := make(chan []byte, 2)
	var apicCoverData, apicIconData []byte
	if base64EncodedApicCoverURI != "" {
		apicCoverURI, err := decodeBase64URI(base64EncodedApicCoverURI)
		if err != nil {
			handleError(&w, err)
			return
		}
		log.Println("Cover URI:", apicCoverURI)
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
		log.Println("Icon URI:", apicIconURI)
		go httpGETChan(apicIconURI, dataChan, errChan)
	} else {
		errChan <- nil
		dataChan <- nil
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
	log.Println("Cover Data len:", len(apicCoverData))
	log.Println("Icon Data len:", len(apicIconData))
	var filename string
	if strings.Contains(decodedURI, ".m3u8") {
		filename, err = download.HLSFile(string(decodedURI), "")
	} else if strings.Contains(decodedURI, ".mp3") {
		filename, err = download.MP3File(string(decodedURI), "")
	} else {
		err = fmt.Errorf("Unsupported file type: %s", filepath.Base(filepath.Dir(decodedURI)))
		handleError(&w, err)
		return
	}
	if err != nil {
		handleError(&w, err)
		return
	}
	defer os.Remove(filename)
	id3File, err := id3.Open(filename)
	if err != nil {
		id3File.Close()
		fileData, err := ioutil.ReadFile(filename)
		if err != nil {
			handleError(&w, err)
			return
		}
		w.Header().Add("Content-Type", "audio/mpeg")
		w.Write(fileData)
		return
	}
	defer id3File.Close()
	utils.SetID3Tag(id3File, performer, title, album, year, trck)
	utils.SetID3TagAPICs(id3File, apicCoverData, apicIconData)
	id3File.Close()
	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		handleError(&w, err)
		return
	}
	w.Header().Add("Content-Type", "audio/mpeg")
	w.Write(fileData)
	log.Println("OK")
	return
}

func main() {
	http.HandleFunc("/", handler)
	iface := ":80"
	log.Printf("Serving on %s\n", iface)
	log.Fatal(http.ListenAndServe(iface, nil))
}
