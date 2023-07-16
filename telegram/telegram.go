package telegram

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
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

	b.RegisterHandler(bot.HandlerTypeMessageText, "https", bot.MatchTypePrefix, yabelineUrlHandler)
	b.Start(ctx)
}

func yabelineUrlHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !strings.HasPrefix(update.Message.Text, "https://yabeline.tw/") {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "wrong url"})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "scrapping stickers. Please wait..."})
	filename, images, isTelegramReady, err := yabeline.GetStickers(update.Message.Text)

	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: fmt.Sprintf("error: %v", err)})
		return
	}

	if len(images) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "Images not found"})
		return
	}

	archivedImages := new(bytes.Buffer)
	zipWriter := zip.NewWriter(archivedImages)
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: update.Message.Chat.ID, Text: "zipping 4 you"})

	for i, image := range images {
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
