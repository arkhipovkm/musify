package utils

import (
	"log"

	"github.com/arkhipovkm/id3-go"
	v2 "github.com/arkhipovkm/id3-go/v2"
)

// SetID3Tag writes these tags to file:
// > performer
// > title
// > Album
// > Year
// > Track number
func SetID3Tag(tag id3.Tagger, performer, title, album, year, trck string) error {
	var err error
	log.Println("Setting ID3 Tag:", performer, title, album, year, trck)
	if performer != "" {
		// fmt.Println("Setting performer: ", performer)
		tag.DeleteFrames(v2.V23CommonFrame["Artist"].Id())
		f := v2.NewTextFrame(v2.V23FrameTypeMap[v2.V23CommonFrame["Artist"].Id()], performer)
		f.SetEncoding("ISO-8859-1")
		tag.AddFrames(f)
	}
	if title != "" {
		// fmt.Println("Setting title: ", title)
		tag.DeleteFrames(v2.V23CommonFrame["Title"].Id())
		f := v2.NewTextFrame(v2.V23FrameTypeMap[v2.V23CommonFrame["Title"].Id()], title)
		f.SetEncoding("ISO-8859-1")
		tag.AddFrames(f)
	}
	if year != "" {
		// fmt.Println("Setting year: ", year)
		tag.DeleteFrames(v2.V23CommonFrame["Year"].Id())
		f := v2.NewTextFrame(v2.V23FrameTypeMap[v2.V23CommonFrame["Year"].Id()], year)
		f.SetEncoding("ISO-8859-1")
		tag.AddFrames(f)
	}
	if album != "" {
		// fmt.Println("Setting album: ", album)
		tag.DeleteFrames(v2.V23CommonFrame["Album"].Id())
		f := v2.NewTextFrame(v2.V23FrameTypeMap[v2.V23CommonFrame["Album"].Id()], album)
		f.SetEncoding("ISO-8859-1")
		tag.AddFrames(f)
	}
	if trck != "" {
		// fmt.Println("Setting trck: ", trck)
		tag.DeleteFrames("TRCK")
		trckFrameType := v2.V23FrameTypeMap["TRCK"]
		f := v2.NewTextFrame(trckFrameType, trck)
		f.SetEncoding("ISO-8859-1")
		tag.AddFrames(f)
	}
	return err
}

// SetID3TagAPICs writes 2 APIC tags:
// > Other Icon (2)
// > Cover(front) (3)
func SetID3TagAPICs(tag id3.Tagger, apicCover []byte, apicIcon []byte) error {
	var err error
	tag.DeleteFrames("APIC")
	frameType := v2.V23FrameTypeMap["APIC"]
	mimeType := "image/jpeg"
	if apicCover != nil {
		log.Println("Setting apicCover: ", len(apicCover))
		apicCoverImageFrame := v2.NewImageFrame(frameType, mimeType, 3, "front cover", apicCover) // 3 for "Cover(front)"
		apicCoverImageFrame.SetEncoding("ISO-8859-1")
		tag.AddFrames(apicCoverImageFrame)
	}
	if apicIcon != nil {
		log.Println("Setting apicIcon: ", len(apicIcon))
		apicIconImageFrame := v2.NewImageFrame(frameType, mimeType, 2, "other icon", apicIcon) // 2 for "Other icon"
		apicIconImageFrame.SetEncoding("ISO-8859-1")
		tag.AddFrames(apicIconImageFrame)
	}
	return err
}
