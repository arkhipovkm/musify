package main

import (
	"log"
	"net/http"
	"os"

	"github.com/arkhipovkm/musify/bot"
	"github.com/arkhipovkm/musify/db"
	"github.com/arkhipovkm/musify/happidev"
	"github.com/arkhipovkm/musify/streamer"
)

func main() {
	defer db.DB.Close()
	herokuAppName := os.Getenv("HEROKU_APP_NAME")
	if herokuAppName == "" {
		panic("No Heroku App Name")
	}
	telegramBotToken := os.Getenv("TELEGRAM_BOT_API_TOKEN")
	if telegramBotToken == "" {
		panic("No Telegram Bot Token")
	}
	telegramOwnerChatID := os.Getenv("TELEGRAM_OWNER_CHAT_ID")
	if telegramOwnerChatID == "" {
		panic("No Telegram Owner Chat ID")
	}
	vkUsername := os.Getenv("VK_USERNAME")
	if vkUsername == "" {
		panic("No VK Username")
	}
	vkPassword := os.Getenv("VK_PASSWORD")
	if vkPassword == "" {
		panic("No VK Password")
	}
	vkAPIAccessToken := os.Getenv("VK_API_ACCESS_TOKEN")
	if vkAPIAccessToken == "" {
		panic("No VK API Access Token")
	}
	musifyDSN := os.Getenv("MUSIFY_SQL_DSN")
	if musifyDSN == "" {
		panic("No Musify MySQL DSN")
	}
	debug := os.Getenv("DEBUG")
	var port string
	if debug == "" {
		log.Println("Running in Production mode.")
		port = os.Getenv("PORT")
		if port == "" {
			panic("No PORT env variable")
		}

	} else {
		port = "8080"
		log.Println("Running in Debug mode")
	}
	go streamer.Streamer()
	go happidev.LyricsServer()
	bot.Bot()

	iface := ":" + port
	log.Printf("Serving on %s\n", iface)
	log.Fatal(http.ListenAndServe(iface, nil))
}
