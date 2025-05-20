package telegram

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
	"yabeline-tg/yabeline"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func StartBot(ctx context.Context, botToken string) {
	opts := []bot.Option{}
	b, err := bot.New(botToken, opts...)

	if err != nil {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{
					URL:  "https://yabeline.tw",
					Text: "yabeline.tw",
				}},
			},
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "Hello, friend. Please send me a sticker url from https://yabeline.tw and I will download them for you.",
			ReplyMarkup: keyboard,
		})
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "https", bot.MatchTypePrefix, yabelineUrlHandler)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.Document != nil && update.Message.Document.MimeType == "application/zip"
	}, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		// send random chat action
		closeChan := make(chan any)
		defer func() { closeChan <- true }()

		go sendRandomChatAction(ctx, b, update, closeChan)

		is_animated := strings.Contains(update.Message.Document.FileName, "animated")
		file, err := b.GetFile(ctx, &bot.GetFileParams{
			FileID: update.Message.Document.FileID,
		})

		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not get file"})
			return
		}

		zipFileResponse, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, file.FilePath))

		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not download file"})
			return
		}

		defer zipFileResponse.Body.Close()

		body, err := io.ReadAll(zipFileResponse.Body)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not read response body"})
			return
		}

		zipFile, err := zip.NewReader(bytes.NewReader(body), int64(zipFileResponse.ContentLength))

		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not read zip file"})
			return
		}

		archivedImages := new(bytes.Buffer)
		zipWriter := zip.NewWriter(archivedImages)
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "converting stickers. Please wait..."})

		for _, f := range zipFile.File {
			if f.FileInfo().IsDir() {
				continue
			}

			file, err := f.Open()

			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not open file"})
				return
			}

			defer file.Close()

			buf := new(bytes.Buffer)
			buf.ReadFrom(file)

			fileExtension := f.Name[strings.LastIndex(f.Name, ".")+1:]

			if fileExtension != "png" {
				b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Only png files are supported"})
				return
			}

			if is_animated {
				fileExtension = "webm"
			}

			var converted []byte

			if is_animated {
				converted, err = yabeline.ConvertApng(buf.Bytes())

				if err != nil {
					b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not convert file"})
					return
				}
			} else {
				converted, err = yabeline.ConvertImage(buf.Bytes())

				if err != nil {
					b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not convert file"})
					return
				}
			}

			zipWriterFile, err := zipWriter.Create(f.Name[0:strings.LastIndex(f.Name, ".")] + "." + fileExtension)

			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not create zip file"})
				return
			}

			_, err = zipWriterFile.Write(converted)

			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not write to zip file"})
				return
			}
		}

		err = zipWriter.Close()

		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Could not close zip file"})
			return
		}

		b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:  update.Message.Chat.ID,
			Caption: "Stickers converted",
			Document: &models.InputFileUpload{
				Data:     archivedImages,
				Filename: fmt.Sprintf("%s_converted.zip", update.Message.Document.FileName[0:strings.LastIndex(update.Message.Document.FileName, ".")]),
			},
		})
	})
	b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{
				Command:     "/start",
				Description: "start",
			},
		},
	})
	b.Start(ctx)
}

var actions = [11]models.ChatAction{
	models.ChatActionTyping,
	models.ChatActionUploadPhoto,
	models.ChatActionRecordVideo,
	models.ChatActionUploadVideo,
	models.ChatActionUploadDocument,
	models.ChatActionFindLocation,
	models.ChatActionRecordVideoNote,
	models.ChatActionUploadVideoNote,
	models.ChatActionRecordVoice,
	models.ChatActionUploadVoice,
	models.ChatActionChooseSticker,
}

func sendRandomChatAction(ctx context.Context, b *bot.Bot, update *models.Update, close chan any) {
	for {

		select {
		case <-close:
			return
		default:
			b.SendChatAction(ctx, &bot.SendChatActionParams{
				ChatID: update.Message.Chat.ID,
				Action: actions[rand.Intn(10)],
			})
			time.Sleep(5 * time.Second)
		}
	}
}

func yabelineUrlHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !strings.HasPrefix(update.Message.Text, "https://yabeline.tw/") {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "wrong url"})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "scrapping stickers. Please wait..."})

	// send random chat action
	closeChan := make(chan any)
	defer func() { closeChan <- true }()

	go sendRandomChatAction(ctx, b, update, closeChan)

	filename, images, isTelegramReady, err := yabeline.GetStickers(update.Message.Text)

	b.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: update.Message.Chat.ID,
		Action: models.ChatActionChooseSticker,
	})

	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: fmt.Sprintf("error: %v", err)})
		return
	}

	if images == nil || len(images) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Images not found"})
		return
	}

	archivedImages := new(bytes.Buffer)
	zipWriter := zip.NewWriter(archivedImages)
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "zipping 4 you"})

	for i, image := range images {
		if image == nil {
			continue
		}

		f, err := zipWriter.Create(fmt.Sprint(i+1) + image.FileExtension)
		if err != nil {
			log.Println(err)
			continue
		}
		_, err = f.Write(image.Data)

		if err != nil {
			log.Println(err)
			continue
		}
	}

	zipWriter.Close()
	caption := "Your stickers here. In that zip file. Go download it and be happy :)"

	if isTelegramReady {
		caption += "\n\nThey are telegram ready btw. You can use some frendly bot and create awesome sticker pack with that sitckers. I could do it myself, but I belive other bots will do it better. Good luck"
	}

	_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: update.Message.Chat.ID,
		Document: &models.InputFileUpload{
			Data:     archivedImages,
			Filename: fmt.Sprintf("%s stickers.zip", filename),
		},
		Caption: caption,
	})

	if err != nil {
		log.Println(err)
	}
}
