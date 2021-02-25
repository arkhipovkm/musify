package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/arkhipovkm/musify/utils"
	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var DB *sql.DB

func init() {
	var err error
	MusifyDSN := os.Getenv("MUSIFY_SQL_DSN")
	if MusifyDSN != "" {
		DB, err = sql.Open(
			"mysql",
			MusifyDSN,
		)
		if err != nil {
			log.Fatal(err)
		}
		err = DB.Ping()
		if err != nil {
			log.Println(err)
		}
	}
}

type Lyrics struct {
	Performer string
	Album     string
	Title     string
	Text      string
	CoverURL  string
}

type Counts struct {
	UsersCount int
	ChatsCount int
	MsgCount   int
	CIRCount   int
}

func GetLyricsByID(id int) (*Lyrics, error) {
	var err error
	var lyrics Lyrics
	if DB == nil {
		return nil, err
	}
	resp := DB.QueryRow("SELECT performer as Performer, album as Album, title as Title, text as Text, cover_url as CoverURL FROM lyrics WHERE id=?", id)
	resp.Scan(&lyrics.Performer, &lyrics.Album, &lyrics.Title, &lyrics.Text, &lyrics.CoverURL)
	return &lyrics, err
}

func GetLyricsByMeta(performer, album, title string) (*Lyrics, error) {
	var err error
	var lyrics Lyrics
	if DB == nil {
		return nil, err
	}
	resp := DB.QueryRow("SELECT * FROM lyrics WHERE performer=? AND album=? AND title=?", performer, album, title)
	resp.Scan(&lyrics)
	return &lyrics, err
}

func GetCounts() (*Counts, error) {
	var counts Counts
	var err error
	if DB == nil {
		return &counts, err
	}
	var resp *sql.Row

	resp = DB.QueryRow("SELECT COUNT(*) as UsersCount FROM users")
	resp.Scan(&counts.UsersCount)

	resp = DB.QueryRow("SELECT COUNT(*) as ChatsCount FROM chats")
	resp.Scan(&counts.ChatsCount)

	resp = DB.QueryRow("SELECT COUNT(*) as MsgCount FROM messages")
	resp.Scan(&counts.MsgCount)

	resp = DB.QueryRow("SELECT COUNT(*) as CIRCount FROM chosen_inline_results")
	resp.Scan(&counts.CIRCount)

	return &counts, err
}

func PutMessage(msg *tgbotapi.Message) error {
	var err error
	if DB == nil {
		return nil
	}
	if msg == nil {
		err = errors.New("Message is nil. Message will not be inserted")
		return err
	}
	if msg.Audio != nil {
		_, err = DB.Exec("INSERT IGNORE INTO audios (file_id, duration, performer, title, mime_type, file_size) VALUES (?, ?, ?, ?, ?, ?)",
			msg.Audio.FileID,
			msg.Audio.Duration,
			msg.Audio.Performer,
			msg.Audio.Title,
			msg.Audio.MimeType,
			msg.Audio.FileSize,
		)
		if err != nil {
			return fmt.Errorf("Error inserting Audio: %s", err)
		}
	}
	if msg.From != nil {
		_, err = DB.Exec("INSERT IGNORE INTO users (id, username, first_name, last_name, language_code, is_bot) VALUES (?, ?, ?, ?, ?, ?)",
			msg.From.ID,
			msg.From.UserName,
			msg.From.FirstName,
			msg.From.LastName,
			msg.From.LanguageCode,
			msg.From.IsBot,
		)
		if err != nil {
			return fmt.Errorf("Error inserting User: %s", err)
		}
	}
	if msg.Chat != nil {
		_, err = DB.Exec("INSERT IGNORE INTO chats (id, type, title, username, first_name, last_name) VALUES (?, ?, ?, ?, ?, ?)",
			msg.Chat.ID,
			msg.Chat.Type,
			msg.Chat.Title,
			msg.Chat.UserName,
			msg.Chat.FirstName,
			msg.Chat.LastName,
		)
		if err != nil {
			return fmt.Errorf("Error inserting Chat: %s", err)
		}
	}
	if msg.Chat != nil && msg.From != nil {
		_, err = DB.Exec("INSERT INTO messages (message_id, date, from_id, chat_id, audio_id) VALUES (?, ?, ?, ?, ?)",
			msg.MessageID,
			msg.Date,
			msg.From.ID,
			msg.Chat.ID,
			msg.Audio.FileID,
		)
		if err != nil {
			return fmt.Errorf("Error inserting Message: %s", err)
		}
	} else {
		log.Println("Message with nil Chat or nil From:")
		utils.LogJSON(msg)
	}
	return err
}

func PutChosenInlineResult(cir *tgbotapi.ChosenInlineResult) error {
	var err error
	if DB == nil {
		return nil
	}
	_, err = DB.Exec("INSERT IGNORE INTO users (id, username, first_name, last_name, language_code, is_bot) VALUES (?, ?, ?, ?, ?, ?)",
		cir.From.ID,
		cir.From.UserName,
		cir.From.FirstName,
		cir.From.LastName,
		cir.From.LanguageCode,
		cir.From.IsBot,
	)
	if err != nil {
		return fmt.Errorf("Error inserting User: %s", err)
	}
	_, err = DB.Exec("INSERT IGNORE INTO chosen_inline_results (from_id, date, query) VALUES (?, ?, ?)",
		cir.From.ID,
		time.Now().Unix(),
		cir.Query,
	)
	if err != nil {
		return fmt.Errorf("Error inserting User: %s", err)
	}
	return err
}

func PutLyrics(performer, album, title, text, coverURL string) (int64, error) {
	var err error
	var id int64
	if DB == nil {
		return id, nil
	}
	result, err := DB.Exec("INSERT INTO lyrics (performer, album, title, text, cover_url) VALUES (?, ?, ?, ?, ?)",
		performer,
		album,
		title,
		text,
		coverURL,
	)
	if err != nil {
		return id, fmt.Errorf("Error inserting Lyrics: %s", err)
	}
	id, err = result.LastInsertId()
	if err != nil {
		return id, err
	}
	return id, err
}

func PutMessageAsync(msg *tgbotapi.Message) {
	err := PutMessage(msg)
	if err != nil {
		log.Println(err)
	}
}

func PutChosenInlineResultAsync(cir *tgbotapi.ChosenInlineResult) {
	err := PutChosenInlineResult(cir)
	if err != nil {
		log.Println(err)
	}
}
