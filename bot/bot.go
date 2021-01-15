package bot

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/arkhipovkm/musify/db"
	"github.com/arkhipovkm/musify/utils"
	"github.com/arkhipovkm/musify/vk"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/uuid"
)

var N_RESULTS int = 10
var vkUser *vk.User = vk.NewDefaultUser()

func vkAuthLoop() {
	for {
		err := vkUser.Authenticate()
		if err != nil {
			log.Panic(err)
		}

		log.Printf("Authenticated on VK Account: %d\n", vkUser.ID)
		time.Sleep(12 * time.Hour)
		utils.ClearCache(vkUser.RemixSID)
	}
}

func prepareAudioStreamURI(a *vk.Audio, album *vk.Playlist) string {
	query := make(url.Values)
	if album != nil {
		var trck string
		if album != nil {
			for i, aa := range album.List {
				if aa.AudioID == a.AudioID {
					trck = strconv.Itoa(i+1) + "/" + strconv.Itoa(album.TotalCount)
					break
				}
			}
		}
		query.Set("trck", trck)
		query.Set("album", album.Title)
		query.Set("year", album.YearInfoStr)
	}
	query.Set("performer", a.Performer)
	query.Set("title", a.Title)
	query.Set("apic_icon", base64.URLEncoding.EncodeToString([]byte(a.CoverURLs)))
	var apicCover string
	if album != nil && album.CoverURL != "" {
		apicCover = album.CoverURL
	} else {
		apicCover = a.CoverURLp
	}
	query.Set("apic_cover", base64.URLEncoding.EncodeToString([]byte(apicCover)))
	return fmt.Sprintf(
		"https://%s.herokuapp.com/streamer/%s/%s.mp3?%s",
		os.Getenv("HEROKU_APP_NAME"),
		base64.URLEncoding.EncodeToString([]byte(a.URL)),
		url.PathEscape(strings.ReplaceAll(a.Performer, "/", "|")+" — "+strings.ReplaceAll(a.Title, "/", "|")),
		query.Encode(),
	)
}

func prepareInlineAudioResult(a *vk.Audio, album *vk.Playlist) interface{} {
	uri := prepareAudioStreamURI(a, album)
	return &tgbotapi.InlineQueryResultAudio{
		Type:      "audio",
		ID:        uuid.New().String(),
		URL:       uri,
		Title:     a.Title,
		Performer: a.Performer,
		Duration:  a.Duration,
	}
}

func getAudioShares(albumID string, u *vk.User) (results []tgbotapi.AudioConfig, err error) {
	defer func() {
		r := recover()
		err, _ = r.(error)
	}()
	playlist := vk.LoadPlaylist(albumID, u)
	playlist.AcquireURLs(u)
	playlist.DecypherURLs(u)
	if playlist == nil {
		return nil, fmt.Errorf("Nil playlist: %s", albumID)
	}

	for _, a := range playlist.List {
		if a.URL != "" {
			uri := prepareAudioStreamURI(a, playlist)
			audioShare := tgbotapi.NewAudioShare(int64(0), uri)
			audioShare.Duration = a.Duration
			audioShare.Performer = a.Performer
			audioShare.Title = a.Title
			results = append(results, audioShare)
		}
	}
	return results, err
}

func getAlbumInlineResults(albumID string, offset int, n int, query string, u *vk.User) (results []interface{}, nextOffset string, err error) {
	nextOffset = strconv.Itoa(offset + n)
	defer func() {
		r := recover()
		err, _ = r.(error)
	}()
	playlist := vk.LoadPlaylist(albumID, u)

	if query != "" {
		var newList []*vk.Audio
		queryRe := regexp.MustCompile("(?i)" + query)
		for _, a := range playlist.List {
			if queryRe.MatchString(a.Performer) || queryRe.MatchString(a.Album) || queryRe.MatchString(a.Title) {
				newList = append(newList, a)
			}
		}
		playlist.List = newList
	}

	if len(playlist.List) < offset {
		playlist.List = nil
	} else {
		playlist.List = playlist.List[offset:]
	}

	if len(playlist.List) > n {
		playlist.List = playlist.List[:n]
	}
	if len(playlist.List) == 0 {
		return
	}
	playlist.AcquireURLs(u)
	playlist.DecypherURLs(u)
	if playlist == nil {
		return results, nextOffset, fmt.Errorf("Nil playlist: %s", albumID)
	}
	for _, a := range playlist.List {
		if a.URL != "" {
			result := prepareInlineAudioResult(a, playlist)
			results = append(results, result)
		}
	}
	return results, nextOffset, err
}

func getSectionInlineResults(query string, offset, n int, u *vk.User) (results []interface{}, nextOffset string, err error) {
	nextOffset = strconv.Itoa(offset + n)
	defer func() {
		r := recover()
		err, _ = r.(error)
	}()
	playlistMap, topPlaylists, audios, err := vk.SectionQuery(query, offset, n, u)
	if err != nil {
		return results, nextOffset, err
	}
	for _, pl := range topPlaylists {
		title := fmt.Sprintf("%s — %s", pl.Title, pl.AuthorName)
		var description string
		if pl.YearInfoStr != "" {
			description = description + pl.YearInfoStr
		}
		if pl.NTracksInfoStr != "" {
			description = description + " - " + pl.NTracksInfoStr
		}
		if pl.NPlaysInfoStr != "" {
			description = description + " - " + pl.NPlaysInfoStr
		}
		var coverSuffix string
		if pl.CoverURL != "" {
			coverSuffix = fmt.Sprintf("[.](%s)", pl.CoverURL)
		}
		inputMessageContent := &tgbotapi.InputTextMessageContent{
			Text:                  title + "\n" + description + coverSuffix,
			ParseMode:             "markdown",
			DisableWebPagePreview: false,
		}
		var id string
		id = pl.FullID()
		// if pl.OwnerID != 0 && pl.ID != 0 && strconv.Itoa(pl.OwnerID)[:2] == "-2" && strings.Contains(strconv.Itoa(pl.OwnerID), strconv.Itoa(pl.ID)) {
		// 	id = utils.Itoa50(pl.ID)
		// } else {
		// 	id = pl.FullID()
		// }
		switchInlineQuery := ":album " + id + " "
		callBackData := "send-all-" + pl.FullID()
		results = append(results, &tgbotapi.InlineQueryResultArticle{
			Type:                "article",
			ID:                  uuid.New().String(),
			Title:               title,
			Description:         description,
			ThumbURL:            pl.CoverURL,
			InputMessageContent: inputMessageContent,
			HideURL:             true,
			ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
					tgbotapi.InlineKeyboardButton{
						Text:                         "Discover",
						SwitchInlineQueryCurrentChat: &switchInlineQuery,
					},
					tgbotapi.InlineKeyboardButton{
						Text:              "Share",
						SwitchInlineQuery: &switchInlineQuery,
					},
					tgbotapi.InlineKeyboardButton{
						Text:         "Get",
						CallbackData: &callBackData,
					},
				}},
			},
		})
	}
	for _, a := range audios {
		if a.URL != "" {
			album := playlistMap[a.Album]
			result := prepareInlineAudioResult(a, album)
			results = append(results, result)
		}
	}
	return results, nextOffset, err
}

func process(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	var err error
	for update := range updates {
		if update.InlineQuery != nil {
			inlineQueryAnswer := tgbotapi.InlineConfig{
				InlineQueryID: update.InlineQuery.ID,
				CacheTime:     3600,
				IsPersonal:    false,
			}
			if update.InlineQuery.Query == "" || update.InlineQuery.Query == " " {
				_, err := bot.AnswerInlineQuery(inlineQueryAnswer)
				if err != nil {
					log.Println(err)
				}
			} else {
				var offset int
				if update.InlineQuery.Offset != "" {
					offset, _ = strconv.Atoi(update.InlineQuery.Offset)
				}
				// Case Album+Query
				re := regexp.MustCompile("^:album (.*?) (.*?)$")
				if re.MatchString(update.InlineQuery.Query) {
					subm := re.FindStringSubmatch(update.InlineQuery.Query)
					albumID := subm[1]
					query := subm[2]
					inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getAlbumInlineResults(albumID, offset, N_RESULTS, query, vkUser)
					if err != nil {
						log.Println(err)
					}
					bot.AnswerInlineQuery(inlineQueryAnswer)
					continue
				}
				// Case Album
				re = regexp.MustCompile("^:album (.*?)$")
				if re.MatchString(update.InlineQuery.Query) {
					subm := re.FindStringSubmatch(update.InlineQuery.Query)
					albumID := subm[1]
					inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getAlbumInlineResults(albumID, offset, N_RESULTS, "", vkUser)
					if err != nil {
						log.Println(err)
					}
					bot.AnswerInlineQuery(inlineQueryAnswer)
					continue
				}
				// Case User+Query
				re = regexp.MustCompile("^@(.*?) (.*?)$")
				if re.MatchString(update.InlineQuery.Query) {
					subm := re.FindStringSubmatch(update.InlineQuery.Query)
					userID := subm[1]
					query := subm[2]
					log.Println("User search + Query:", query, ".")
					searcheeUsers, err := vk.UsersGet(userID)
					if err != nil {
						log.Println(err)
						bot.AnswerInlineQuery(inlineQueryAnswer)
						continue
					}
					if searcheeUsers == nil || len(searcheeUsers) == 0 {
						log.Println("No such VK user:", userID)
						bot.AnswerInlineQuery(inlineQueryAnswer)
						continue
					}
					searcheeUser := searcheeUsers[0]
					albumID := fmt.Sprintf("%d_-1", searcheeUser.ID)
					log.Printf("Converted the %s user into its main album id: %s", userID, albumID)
					inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getAlbumInlineResults(albumID, offset, N_RESULTS, query, vkUser)
					if err != nil {
						log.Println(err)
					}
					bot.AnswerInlineQuery(inlineQueryAnswer)
					continue
				}
				// Case user
				re = regexp.MustCompile("^@(.*?)$")
				if re.MatchString(update.InlineQuery.Query) {
					subm := re.FindStringSubmatch(update.InlineQuery.Query)
					userID := subm[1]
					searcheeUsers, err := vk.UsersGet(userID)
					if err != nil {
						log.Println(err)
						bot.AnswerInlineQuery(inlineQueryAnswer)
						continue
					}
					if searcheeUsers == nil || len(searcheeUsers) == 0 {
						log.Println("No such VK user:", userID)
						bot.AnswerInlineQuery(inlineQueryAnswer)
						continue
					}
					searcheeUser := searcheeUsers[0]
					albumID := fmt.Sprintf("%d_-1", searcheeUser.ID)
					log.Printf("Converted the %s user into its main album id: %s", userID, albumID)
					inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getAlbumInlineResults(albumID, offset, N_RESULTS, "", vkUser)
					if err != nil {
						log.Println(err)
					}
					bot.AnswerInlineQuery(inlineQueryAnswer)
					continue
				}
				inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getSectionInlineResults(update.InlineQuery.Query, offset, N_RESULTS, vkUser)
				if err != nil {
					log.Println(err)
				}
				bot.AnswerInlineQuery(inlineQueryAnswer)
				continue
			}
		} else if update.CallbackQuery != nil {
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			re := regexp.MustCompile("send-all-(.*?)$")
			if re.MatchString(update.CallbackQuery.Data) {
				subm := re.FindStringSubmatch(update.CallbackQuery.Data)
				albumID := subm[1]
				audioShares, err := getAudioShares(albumID, vkUser)
				if err != nil {
					log.Println(err)
				}
				var chatID int64
				if update.CallbackQuery.Message != nil &&
					update.CallbackQuery.Message.Chat != nil &&
					update.CallbackQuery.Message.Chat.ID != 0 {
					chatID = update.CallbackQuery.Message.Chat.ID
				} else if update.CallbackQuery.From != nil {
					chatID = int64(update.CallbackQuery.From.ID)
				} else {
					continue
				}
				for _, audioShare := range audioShares {
					audioShare.ChatID = chatID
					msg, _ := bot.Send(audioShare)
					if msg.MessageID != 0 {
						go db.PutMessageAsync(&msg)
						// utils.LogJSON(&msg)
					}
				}
			}
		} else if update.Message != nil {
			if update.Message.IsCommand() {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Use inline query to search music")
				switch update.Message.Command() {
				case "help":
					msg.Text = "Use this bot in inline mode in any chat to search and send music"
					switchInlineQuery := ""
					msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
						InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
							tgbotapi.InlineKeyboardButton{
								Text:                         "Start",
								SwitchInlineQueryCurrentChat: &switchInlineQuery,
							},
						}},
					}
					bot.Send(msg)
				case "stats":
					msg.Text = fmt.Sprintf("Cache writes: %d. Cache Reads: %d\nVK Requests: %d. VK Errors: %d, VK Auths: %d",
						atomic.LoadUint64(&utils.CacheWriteAccessCounter),
						atomic.LoadUint64(&utils.CacheReadAccessCounter),
						atomic.LoadUint64(&vk.VKRequestCounter),
						atomic.LoadUint64(&vk.VKErrorCounter),
						atomic.LoadUint64(&vk.VKAuthCounter),
					)
					bot.Send(msg)
				}
			} else if update.Message.Audio != nil {
				go db.PutMessageAsync(update.Message)
				// utils.LogJSON(update.Message)
			}
		} else if update.ChosenInlineResult != nil {
			go db.PutChosenInlineResult(update.ChosenInlineResult)
			// utils.LogJSON(update.ChosenInlineResult)
		}
	}
}

func Bot() {

	go vkAuthLoop()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_API_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("Authenticated on Telegram Bot account %s", bot.Self.UserName)

	_, err = bot.SetWebhook(tgbotapi.NewWebhook(fmt.Sprintf("https://%s.herokuapp.com/%s", os.Getenv("HEROKU_APP_NAME"), bot.Token)))
	if err != nil {
		log.Fatal(err)
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}
	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}
	updates := bot.ListenForWebhook("/" + bot.Token)
	// _, err = bot.RemoveWebhook()
	// u := tgbotapi.NewUpdate(0)
	// u.Timeout = 60
	// updates, err := bot.GetUpdatesChan(u)
	for w := 0; w < runtime.NumCPU()+2; w++ {
		go process(bot, updates)
	}
}
