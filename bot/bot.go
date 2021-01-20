package bot

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/AudDMusic/audd-go"
	"github.com/arkhipovkm/musify/db"
	"github.com/arkhipovkm/musify/lyrics"
	"github.com/arkhipovkm/musify/utils"
	"github.com/arkhipovkm/musify/vk"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/uuid"
)

var N_RESULTS int = 10
var vkUser *vk.User = vk.NewDefaultUser()

var CaptchaSID string
var CaptchaKey string

func vkAuthLoop() {
	for {
		err := vkUser.Authenticate(CaptchaSID, CaptchaKey)
		if err != nil {
			log.Println(err)
		}
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

func prepareInlineAudioResult(a *vk.Audio, album *vk.Playlist) *tgbotapi.InlineQueryResultAudio {
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

// InlineReAlbumAndQuery - Case Album+Query
var InlineReAlbumAndQuery *regexp.Regexp = regexp.MustCompile("^:album (.*?) (.*?)$")

// InlineReAlbum - Case Album
var InlineReAlbum *regexp.Regexp = regexp.MustCompile("^:album (.*?)$")

// InlineReUserAndQuery - Case User+Query
var InlineReUserAndQuery *regexp.Regexp = regexp.MustCompile("^@(.*?) (.*?)$")

// InlineReUser - Case user
var InlineReUser *regexp.Regexp = regexp.MustCompile("^@(.*?)$")

func process(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	var err error
	for update := range updates {
		if update.InlineQuery != nil {
			inlineQueryAnswer := tgbotapi.InlineConfig{
				InlineQueryID: update.InlineQuery.ID,
				CacheTime:     3600,
				IsPersonal:    false,
			}
			// if update.InlineQuery.Query == "" || update.InlineQuery.Query == " " {
			// 	_, err := bot.AnswerInlineQuery(inlineQueryAnswer)
			// 	if err != nil {
			// 		log.Println(err)
			// 	}
			// 	continue
			// } else {
			var offset int
			if update.InlineQuery.Offset != "" {
				offset, _ = strconv.Atoi(update.InlineQuery.Offset)
			}
			if InlineReAlbumAndQuery.MatchString(update.InlineQuery.Query) {
				parts := InlineReAlbumAndQuery.FindStringSubmatch(update.InlineQuery.Query)
				albumID := parts[1]
				query := parts[2]
				inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getAlbumInlineResults(albumID, offset, N_RESULTS, query, vkUser)
				if err != nil {
					log.Println(err)
				}
				_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
				if err != nil {
					log.Println(err)
				}
				continue
			} else if InlineReAlbum.MatchString(update.InlineQuery.Query) {
				parts := InlineReAlbum.FindStringSubmatch(update.InlineQuery.Query)
				albumID := parts[1]
				inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getAlbumInlineResults(albumID, offset, N_RESULTS, "", vkUser)
				if err != nil {
					log.Println(err)
				}
				_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
				if err != nil {
					log.Println(err)
				}
				continue
			} else if InlineReUserAndQuery.MatchString(update.InlineQuery.Query) {
				parts := InlineReUserAndQuery.FindStringSubmatch(update.InlineQuery.Query)
				userID := parts[1]
				query := parts[2]
				log.Println("User search + Query:", query, ".")
				searcheeUsers, err := vk.UsersGet(userID)
				if err != nil {
					log.Println(err)
					_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
					if err != nil {
						log.Println(err)
					}
					continue
				}
				if searcheeUsers == nil || len(searcheeUsers) == 0 {
					log.Println("No such VK user:", userID)
					_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
					if err != nil {
						log.Println(err)
					}
					continue
				}
				searcheeUser := searcheeUsers[0]
				albumID := fmt.Sprintf("%d_-1", searcheeUser.ID)

				log.Printf("Converted the %s user into its main album id: %s", userID, albumID)

				inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getAlbumInlineResults(albumID, offset, N_RESULTS, query, vkUser)
				if err != nil {
					log.Println(err)
				}
				_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
				if err != nil {
					log.Println(err)
				}
				continue
			} else if InlineReUser.MatchString(update.InlineQuery.Query) {
				parts := InlineReUser.FindStringSubmatch(update.InlineQuery.Query)
				userID := parts[1]
				searcheeUsers, err := vk.UsersGet(userID)
				if err != nil {
					log.Println(err)
					_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
					if err != nil {
						log.Println(err)
					}
					continue
				}
				if searcheeUsers == nil || len(searcheeUsers) == 0 {
					log.Println("No such VK user:", userID)
					_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
					if err != nil {
						log.Println(err)
					}
					continue
				}
				searcheeUser := searcheeUsers[0]
				albumID := fmt.Sprintf("%d_-1", searcheeUser.ID)
				log.Printf("Converted the %s user into its main album id: %s", userID, albumID)
				inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getAlbumInlineResults(albumID, offset, N_RESULTS, "", vkUser)
				if err != nil {
					log.Println(err)
				}
				_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
				if err != nil {
					log.Println(err)
				}
				continue
			} else {
				inlineQueryAnswer.Results, inlineQueryAnswer.NextOffset, err = getSectionInlineResults(update.InlineQuery.Query, offset, N_RESULTS, vkUser)
				if err != nil {
					log.Println(err)
				}
				_, err = bot.AnswerInlineQuery(inlineQueryAnswer)
				if err != nil {
					log.Println(err)
				}
				continue
			}
			// }
		} else if update.CallbackQuery != nil {
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			var re *regexp.Regexp
			re = regexp.MustCompile("^send-all-(.*?)$")
			if re.MatchString(update.CallbackQuery.Data) {
				parts := re.FindStringSubmatch(update.CallbackQuery.Data)
				albumID := parts[1]
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
					msg, err := bot.Send(audioShare)
					if err != nil {
						log.Println(err)
					}
					if msg.MessageID != 0 {
						go db.PutMessageAsync(&msg)
						// utils.LogJSON(&msg)
					}
				}
			}
			re = regexp.MustCompile("^lyrics-(\\d+)-(\\d+)-(\\d+)-(\\d+)$")
			if re.MatchString(update.CallbackQuery.Data) {
				subm := re.FindStringSubmatch(update.CallbackQuery.Data)
				idArtist, err := strconv.Atoi(subm[1])
				if err != nil {
					log.Println(err)
					continue
				}
				idAlbum, err := strconv.Atoi(subm[2])
				if err != nil {
					log.Println(err)
					continue
				}
				idTrack, err := strconv.Atoi(subm[3])
				if err != nil {
					log.Println(err)
					continue
				}
				replyToMessageID, err := strconv.Atoi(subm[4])
				if err != nil {
					log.Println(err)
					continue
				}
				lyrics, err := lyrics.HappiGetLyrics(idArtist, idAlbum, idTrack)
				if err != nil {
					log.Println(err)
					continue
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
				msg := tgbotapi.NewMessage(chatID, lyrics.Lyrics)
				msg.ReplyToMessageID = replyToMessageID
				bot.Send(msg)
			}
		} else if update.Message != nil {
			if update.Message.IsCommand() {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Use inline query to search music")
				switch update.Message.Command() {
				case "help":
					msg.Text = "Use this bot in inline mode in any chat to search and send music.\n\nReply to any audio to open its menu and lyrics.\n\nVK Users, in inline mode type `@<your VK ID or nickname>`, e.g. `@durov` or `@123456` to open your VK music (Note that your VK music must be open to public).\n\nThis bot is open-source and is available on [GitHub](https://github.com/arkhipovkm/musify). Anyone can run a copy of it on its own server 👩‍💻🧑‍💻. Visit [home page](https://github.com/arkhipovkm/musify) for instructions!"
					msg.DisableWebPagePreview = true
					msg.ParseMode = "markdown"
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
					counts, err := db.GetCounts()
					if err != nil {
						log.Println(err)
					}
					msg.Text = fmt.Sprintf("Cache writes: %d. Cache Reads: %d\nVK Requests: %d. VK Errors: %d, VK Auths: %d\nUsers: %d, Chats: %d, Messages: %d, CIRs: %d",
						atomic.LoadUint64(&utils.CacheWriteAccessCounter),
						atomic.LoadUint64(&utils.CacheReadAccessCounter),
						atomic.LoadUint64(&vk.VKRequestCounter),
						atomic.LoadUint64(&vk.VKErrorCounter),
						atomic.LoadUint64(&vk.VKAuthCounter),
						counts.UsersCount,
						counts.ChatsCount,
						counts.MsgCount,
						counts.CIRCount,
					)
					bot.Send(msg)
				}
			} else if update.Message.Audio != nil {
				go db.PutMessageAsync(update.Message)
				// utils.LogJSON(update.Message)
			} else if update.Message.Voice != nil {
				if os.Getenv("AUDD_API_TOKEN") != "" {

					fileConfig := tgbotapi.FileConfig{
						FileID: update.Message.Voice.FileID,
					}
					file, err := bot.GetFile(fileConfig)
					if err != nil {
						log.Println(err)
						continue
					}

					fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", os.Getenv("TELEGRAM_BOT_API_TOKEN"), file.FilePath)
					log.Println(fileURL)

					resp, err := http.Get(fileURL)
					if err != nil {
						log.Println(err)
						continue
					}
					defer resp.Body.Close()

					client := audd.NewClient(os.Getenv("AUDD_API_TOKEN"))
					auddResp, err := client.RecognizeByFile(resp.Body, "lyrics,spotify", nil)
					if err != nil {
						log.Println(err)
						continue
					}

					// body, err := ioutil.ReadFile("audd_response.json")
					// if err != nil {
					// 	log.Println(err)
					// 	continue
					// }
					// var auddResp audd.RecognitionResult
					// json.Unmarshal(body, &auddResp)

					utils.LogJSON(auddResp)

					query := auddResp.Artist + " " + auddResp.Title
					playlistMap, _, audios, err := vk.SectionQuery(query, 0, 1, vkUser)
					if err != nil {
						log.Println(err)
						continue
					}

					if len(audios) > 0 {
						a := audios[0]
						playlist := playlistMap[a.Album]

						uri := prepareAudioStreamURI(a, playlist)
						audioShare := tgbotapi.NewAudioShare(int64(0), uri)
						audioShare.Duration = a.Duration
						audioShare.Performer = a.Performer
						audioShare.Title = a.Title
						audioShare.ChatID = update.Message.Chat.ID
						audioShare.ReplyToMessageID = update.Message.MessageID

						audioMsg, err := bot.Send(audioShare)
						if err != nil {
							log.Println(err)
							continue
						}

						encodedLyrics := base64.URLEncoding.EncodeToString([]byte(auddResp.Lyrics.Lyrics))
						encodedCoverURL := base64.URLEncoding.EncodeToString([]byte(auddResp.Spotify.Album.Images[0].URL))
						encodedArtist := base64.URLEncoding.EncodeToString([]byte(auddResp.Artist))
						encodedTrack := base64.URLEncoding.EncodeToString([]byte(auddResp.Title))
						lyricsURL := fmt.Sprintf("https://%s.herokuapp.com/alyrics/%s/%s/%s/%s", os.Getenv("HEROKU_APP_NAME"), encodedLyrics, encodedCoverURL, encodedArtist, encodedTrack)
						log.Println(lyricsURL)

						msg := tgbotapi.NewMessage(
							update.Message.Chat.ID,
							fmt.Sprintf("%s — %s", a.Performer, a.Title),
						)
						msg.ReplyToMessageID = audioMsg.MessageID
						switchInlineQuery := a.Performer + " "
						msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
							InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
								tgbotapi.InlineKeyboardButton{
									Text:                         "Search Artist",
									SwitchInlineQueryCurrentChat: &switchInlineQuery,
								},
							}},
						}
						lyricsIVURL := fmt.Sprintf("https://t.me/iv?url=%s&rhash=%s", url.PathEscape(lyricsURL), os.Getenv("TELEGRAM_RHASH"))
						msg.Text = fmt.Sprintf("[%s — %s](%s)", a.Performer, a.Title, lyricsIVURL)
						msg.ParseMode = "markdown"
						bot.Send(&msg)
					}
					continue
				}
			} else if update.Message.ReplyToMessage != nil {
				reCaptchaURL := regexp.MustCompile("\\?sid=(.*?)$")
				if update.Message.ReplyToMessage.Entities != nil {
					entities := *update.Message.ReplyToMessage.Entities
					var ent tgbotapi.MessageEntity
					if len(entities) == 0 {
						continue
					} else {
						ent = entities[0]
					}
					if reCaptchaURL.MatchString(ent.URL) {
						parts := reCaptchaURL.FindStringSubmatch(ent.URL)
						CaptchaSID = parts[1]
						CaptchaKey = update.Message.Text
						log.Println("Received captcha SID and Key:", CaptchaSID, CaptchaKey)
						utils.ClearCache(vkUser.RemixSID)
						err = vkUser.Authenticate(CaptchaSID, CaptchaKey)
						var msg tgbotapi.MessageConfig
						if err == nil {
							msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Successful login 💪")
							msg.ReplyToMessageID = update.Message.MessageID
							bot.Send(&msg)
						}
					}
				} else if update.Message.ReplyToMessage.Audio != nil {
					audio := update.Message.ReplyToMessage.Audio
					msg := tgbotapi.NewMessage(
						update.Message.Chat.ID,
						fmt.Sprintf("%s — %s", audio.Performer, audio.Title),
					)
					msg.ReplyToMessageID = update.Message.ReplyToMessage.MessageID
					switchInlineQuery := audio.Performer + " "
					msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
						InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
							tgbotapi.InlineKeyboardButton{
								Text:                         "Search Artist",
								SwitchInlineQueryCurrentChat: &switchInlineQuery,
							},
						}},
					}

					q := fmt.Sprintf("%s %s", audio.Performer, audio.Title)
					searchResults, err := lyrics.HappiSearch(q)
					if err != nil {
						log.Println(err)
						// bot.Send(msg)
						// continue
					}
					utils.LogJSON(searchResults)
					bestResult, err := lyrics.HappiFindBestMatch(audio.Performer, audio.Title, searchResults)
					if err != nil {
						log.Println(err)
					}
					var lyricsURL string
					if bestResult != nil && bestResult.HasLyrics {
						lyricsURL = fmt.Sprintf("https://%s.herokuapp.com/hlyrics/%d/%d/%d", os.Getenv("HEROKU_APP_NAME"), bestResult.IDArtist, bestResult.IDAlbum, bestResult.IDTrack)
					} else {
						client := audd.NewClient(os.Getenv("AUDD_API_TOKEN"))
						foundLyrics, err := client.FindLyrics(q, nil)
						if err != nil {
							log.Println(err)
							continue
						}
						if len(foundLyrics) == 0 {
							log.Println("Empty Audd Lyrics")
							continue
						}
						bestFoundLyrics := foundLyrics[0]
						encodedLyrics := base64.URLEncoding.EncodeToString([]byte(bestFoundLyrics.Lyrics))
						encodedArtist := base64.URLEncoding.EncodeToString([]byte(bestFoundLyrics.Artist))
						encodedTrack := base64.URLEncoding.EncodeToString([]byte(bestFoundLyrics.Title))

						var encodedCoverURL string
						if bestResult != nil {
							albumID := strconv.Itoa(bestResult.IDAlbum)
							encodedCoverURL = base64.URLEncoding.EncodeToString([]byte("https://api.happi.dev/v1/music/cover/" + albumID))
						} else {
							encodedCoverURL = "_"
						}

						lyricsURL = fmt.Sprintf("https://%s.herokuapp.com/alyrics/%s/%s/%s/%s", os.Getenv("HEROKU_APP_NAME"), encodedLyrics, encodedCoverURL, encodedArtist, encodedTrack)
					}

					rHash := os.Getenv("TELEGRAM_RHASH")
					if rHash != "" {
						lyricsIVURL := fmt.Sprintf("https://t.me/iv?url=%s&rhash=%s", url.PathEscape(lyricsURL), rHash)
						msg.Text = fmt.Sprintf("[%s — %s](%s)", audio.Performer, audio.Title, lyricsIVURL)
						msg.ParseMode = "markdown"
					} else {
						callBackData := fmt.Sprintf("lyrics-%d-%d-%d-%d", bestResult.IDArtist, bestResult.IDAlbum, bestResult.IDTrack, update.Message.ReplyToMessage.MessageID)
						msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
							InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
								tgbotapi.InlineKeyboardButton{
									Text:         "Lyrics",
									CallbackData: &callBackData,
								},
								tgbotapi.InlineKeyboardButton{
									Text:                         "Search Artist",
									SwitchInlineQueryCurrentChat: &switchInlineQuery,
								},
							}},
						}
					}
					bot.Send(msg)
				}
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

	debug := false
	debugEnv := os.Getenv("DEBUG")
	if debugEnv != "" {
		debug = true
	}

	bot.Debug = debug
	log.Printf("Authenticated on Telegram Bot account %s", bot.Self.UserName)

	var updates tgbotapi.UpdatesChannel
	if !debug {
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
		updates = bot.ListenForWebhook("/" + bot.Token)
	} else {
		_, err = bot.RemoveWebhook()
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		updates, err = bot.GetUpdatesChan(u)
	}

	for w := 0; w < runtime.NumCPU()+2; w++ {
		go process(bot, updates)
	}
}
