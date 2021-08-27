package download

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/arkhipovkm/musify/utils"
	"github.com/arkhipovkm/musify/vk"
)

func Download(audio *vk.Audio, filename string) error {
	var err error
	var n int
	if strings.Contains(audio.URL, ".m3u8") {
		re := regexp.MustCompile("/[0-9a-f]+(/audios)?/([0-9a-f]+)/index.m3u8")
		replacedAudioURL := re.ReplaceAllString(audio.URL, "$1/$2.mp3")
		if audio.URL == replacedAudioURL {
			log.Println("Failed to replace HLS to MP3. Proceeding in HLS mode (requires ffmpeg)")
			_, n, err = HLSFile(audio.URL, filename)
		} else {
			_, n, err = MP3File(replacedAudioURL, filename)
		}
	} else if strings.Contains(audio.URL, ".mp3") {
		_, n, err = MP3File(audio.URL, filename)
	} else {
		err = fmt.Errorf("Unsupported file type: %s", filepath.Base(filepath.Dir(audio.URL)))
	}
	log.Printf("Downloaded %d bytes\n", n)
	return err
}

func DownloadAPICs(audio *vk.Audio, album *vk.Playlist) ([]byte, []byte) {
	apicErrChan := make(chan error, 2)
	apicDataChan := make(chan []byte, 2)
	var apicCoverData, apicIconData []byte

	var apicCover string
	if album != nil && album.CoverURL != "" {
		apicCover = album.CoverURL
	} else {
		apicCover = audio.CoverURLp
	}

	if apicCover != "" {
		log.Println("Fetching apic cover (large):", apicCover)
		go utils.HttpGETChan(apicCover, apicDataChan, apicErrChan)
	} else {
		apicErrChan <- nil
		apicDataChan <- nil
	}
	if audio.CoverURLs != "" {
		log.Println("Fetching apic icon (small):", audio.CoverURLs)
		go utils.HttpGETChan(audio.CoverURLs, apicDataChan, apicErrChan)
	} else {
		apicErrChan <- nil
		apicDataChan <- nil
	}

	for i := 0; i < 2; i++ {
		err := <-apicErrChan
		if err != nil {
			log.Println("Error loading APICs: ", err)
		}
	}

	apic0 := <-apicDataChan
	apic1 := <-apicDataChan

	if len(apic0) < len(apic1) {
		apicCoverData = apic1
		apicIconData = apic0
	} else {
		apicCoverData = apic0
		apicIconData = apic1
	}
	return apicCoverData, apicIconData
}
