package main

import (
	"log"
	"net/http"
	"os"

	"github.com/arkhipovkm/musify/bot"
	"github.com/arkhipovkm/musify/db"
	"github.com/arkhipovkm/musify/server"
)

func main() {
	defer db.DB.Close()
	appHostname := os.Getenv("APP_HOSTNAME")
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
	telegramOwnerChatID := os.Getenv("TELEGRAM_OWNER_CHAT_ID")
	if telegramOwnerChatID == "" {
		panic("No Telegram Owner Chat ID")
	}
	vkAPIAccessToken := os.Getenv("VK_API_ACCESS_TOKEN")
	if vkAPIAccessToken == "" {
		log.Println("WARNING. No VK API Access Token")
	}
	musifyDSN := os.Getenv("MUSIFY_SQL_DSN")
	if musifyDSN == "" {
		log.Println("WARNING. No Musify MySQL DSN")
	}
	happiDev := os.Getenv("HAPPIDEV_API_TOKEN")
	if happiDev == "" {
		log.Println("WARNING. No HappiDev API Token. Lyrics will be unavailable.")
	}
	auddToken := os.Getenv("AUDD_API_TOKEN")
	if auddToken == "" {
		log.Println("WARNING. No AUDD API Token. Voice music recognition and some lyrics will be unavailable.")
	}
	tgRhash := os.Getenv("TELEGRAM_RHASH")
	if tgRhash == "" {
		log.Println("WARNING. No Telegram InstantView RHash. Lyrics will not be available thru InstantView")
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
		port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Println("Running in Debug mode")
	}

	go server.ServeMusic()
	go server.ServeTgFiles()
	go server.ServeHappiDevLyrics()
	go server.ServeAuddDirectLyrics()
	go server.ServeIDLyrics()

	bot.Bot()

	iface := ":" + port
	log.Printf("Serving on %s\n", iface)
	log.Fatal(http.ListenAndServe(iface, nil))
}
