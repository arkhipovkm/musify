package utils

import (
	"log"

	"github.com/bogem/id3v2"
)

// SetID3Tag writes these tags to file:
// > performer
// > title
// > Album
// > Year
// > Track number
func SetID3Tag(tag *id3v2.Tag, performer, title, album, year, trck string) error {
	var err error
	log.Println("Setting ID3 Tag:", performer, title, album, year, trck)

	if performer != "" {
		log.Println("Setting performer: ", performer)
		tag.SetArtist(performer)
	}
	if title != "" {
		log.Println("Setting title: ", title)
		tag.SetTitle(title)
	}
	if year != "" {
		log.Println("Setting year: ", year)
		tag.SetYear(year)
	}
	if album != "" {
		log.Println("Setting album: ", album)
		tag.SetAlbum(album)
	}
	if trck != "" {
		log.Println("Setting trck: ", trck)
		tag.AddTextFrame("TRCK", tag.DefaultEncoding(), trck)
	}
	return err
}

// SetID3TagAPICs writes 2 APIC tags:
// > Other Icon (2)
// > Cover(front) (3)
func SetID3TagAPICs(tag *id3v2.Tag, apicCover []byte, apicIcon []byte) error {
	var err error

	if apicIcon != nil {
		log.Println("Setting apicIcon: ", len(apicIcon))
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    tag.DefaultEncoding(),
			MimeType:    "image/jpeg",
			PictureType: 2,
			Description: "other icon",
			Picture:     apicIcon,
		})
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    tag.DefaultEncoding(),
			MimeType:    "image/jpeg",
			PictureType: 3,
			Description: "front cover",
			Picture:     apicCover,
		})
	}
	return err
}
