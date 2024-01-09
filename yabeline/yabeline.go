package yabeline

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type YabelineSticker struct {
	Data          []byte
	FileExtension string
}

func modifyUrlToHighResolution(url string) string {
	if strings.Contains(url, "@2x") || strings.Contains(url, "iphone") {
		return url
	}

	highResUrl := strings.ReplaceAll(url, "android", "ios")
	indexOfDot := strings.LastIndex(highResUrl, ".")

	return highResUrl[:indexOfDot] + "@2x" + highResUrl[indexOfDot:]
}

func modifuApngUrlToHighResolution(url string) string {
	if strings.Contains(url, "/IOS/") {
		return strings.ReplaceAll(url, "IOS", "android")
	}

	return url
}

func extractStickersFromHtmlNode(doc *goquery.Document) (string, []*YabelineSticker, bool, error) {
	if doc == nil {
		return "", nil, false, fmt.Errorf("No document to extract Stickers from")
	}

	imgs := doc.Find("body .stickerBlock").First().Find("img")
	title := doc.Find("body .stickerData").First().Find(".title").Text()
	stickers := make([]*YabelineSticker, len(imgs.Nodes))
	isTelegramReady := true
	var wg sync.WaitGroup
	wg.Add(len(imgs.Nodes))
	var mutex sync.Mutex
	downloadImg := func(i int, node *goquery.Selection) {
		defer wg.Done()
		var imgLink string
		dataAnim, existsAnim := node.Attr("data-anim")
		apng, existsApng := node.Attr("data-apng-src")
		imgType := "static"

		if !existsAnim && !existsApng {
			src, srcExists := node.Attr("src")

			if !srcExists {
				return
			}

			imgLink = modifyUrlToHighResolution(src)
		} else if existsAnim {
			imgType = "apng"
			imgLink = modifuApngUrlToHighResolution(dataAnim)
		} else if existsApng {
			imgType = "apng"
			imgLink = modifuApngUrlToHighResolution(apng)
		}

		res, err := http.Get(imgLink)

		if err != nil {
			return
		}

		defer res.Body.Close()
		img, err := io.ReadAll(res.Body)

		if err != nil {
			return
		}

		lastDot := strings.LastIndex(imgLink, ".")
		fileExtension := imgLink[lastDot:]

		switch imgType {
		case "apng":
			convertedImage, err := ConvertApng(img)

			if err != nil {
				isTelegramReady = false
				log.Println(err)
				break
			}

			img = convertedImage
			fileExtension = ".webm"
		case "static":
			fallthrough
		default:
			convertedImage, err := ConvertImage(img)
			if err != nil {
				isTelegramReady = false
				log.Println(err)
				break
			}

			img = convertedImage
		}
		mutex.Lock()
		stickers[i] = &YabelineSticker{
			Data:          img,
			FileExtension: fileExtension,
		}
		mutex.Unlock()
	}

	imgs.Each(func(i int, node *goquery.Selection) {
		go downloadImg(i, node)
	})

	wg.Wait()
	return title, stickers, isTelegramReady, nil
}

func GetStickers(url string) (packName string, images []*YabelineSticker, isTelegramReady bool, err error) {
	if !strings.HasPrefix(url, "https://yabeline.tw/Stickers_Data.php") {
		return "", nil, false, fmt.Errorf("Invalid URL: %s\nExpected: https://yabeline.tw/Stickers_Data.php?Number=[id]", url)
	}

	res, err := http.Get(url)

	if err != nil {
		return "", nil, false, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", nil, false, fmt.Errorf("HTTP error: %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)

	if err != nil {
		return "", nil, false, err
	}

	return extractStickersFromHtmlNode(doc)
}
