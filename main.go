package main

import (
	"os"

	"github.com/arkhipovkm/musify/bot"
	"github.com/arkhipovkm/musify/streamer"
)

func main() {
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
}