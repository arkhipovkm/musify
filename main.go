package main

import (
	"log"
	"net/http"
	"os"

	"github.com/arkhipovkm/musify/bot"
	"github.com/arkhipovkm/musify/db"
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
	vkUsername := os.Getenv("VK_USERNAME")
	if vkUsername == "" {
		panic("No VK Username")
	}
	vkPassword := os.Getenv("VK_PASSWORD")
	if vkPassword == "" {
		panic("No VK Password")
	}

	go streamer.Streamer()
	bot.Bot()

	iface := ":" + os.Getenv("PORT")
	log.Printf("Serving on %s\n", iface)
	log.Fatal(http.ListenAndServe(iface, nil))
}
