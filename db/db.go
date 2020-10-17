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
	MusifyDSN := os.Getenv("MUSIFY_DSN")
	log.Println(MusifyDSN)
	DB, err = sql.Open(
		"mysql",
		MusifyDSN,
	)
	if err != nil {
		log.Fatal(err)
	}
	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func PutMessage(msg *tgbotapi.Message) error {
	var err error
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
