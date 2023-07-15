package yabeline

import (
	"fmt"
	"io"
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

	highResUrl := strings.Replace(url, "android", "ios", -1)
	indexOfDot := strings.LastIndex(highResUrl, ".")
	return highResUrl[:indexOfDot] + "@2x" + highResUrl[indexOfDot:]

}

func extractStickersFromHtmlNode(doc *goquery.Document) (string, []*YabelineSticker, error) {
	if doc == nil {
		return "", nil, fmt.Errorf("No document to extract Stickers from")
	}

	imgs := doc.Find("body .stickerBlock").Find("img")
	title := doc.Find("body .stickerData").First().Find(".title").Text()
	stickers := make([]*YabelineSticker, len(imgs.Nodes))
	var wg sync.WaitGroup
	wg.Add(len(imgs.Nodes))
	var mutex sync.Mutex
	downloadImg := func(i int, node *goquery.Selection) {
		defer wg.Done()
		var imgLink string
		dataAnim, existsAnim := node.Attr("data-anim")
		apng, existsApng := node.Attr("data-apng-src")

		if !existsAnim && !existsApng {
			src, srcExists := node.Attr("src")

			if !srcExists {
				return
			}

			imgLink = modifyUrlToHighResolution(src)
		} else if existsAnim {
			imgLink = dataAnim
		} else if existsApng {
			imgLink = apng
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
		mutex.Lock()
		stickers[i] = &YabelineSticker{
			Data:          img,
			FileExtension: imgLink[lastDot:],
		}
		mutex.Unlock()
	}
	imgs.Each(func(i int, node *goquery.Selection) {
		go downloadImg(i, node)
	})
	wg.Wait()
	return title, stickers, nil
}

func GetStickers(url string) (packName string, images []*YabelineSticker, err error) {
	if !strings.HasPrefix(url, "https://yabeline.tw/Emoji_Data") {
		return "", nil, fmt.Errorf("Invalid URL: %s", url)
	}

	res, err := http.Get(url)

	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("HTTP error: %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", nil, err
	}

	return extractStickersFromHtmlNode(doc)
}
