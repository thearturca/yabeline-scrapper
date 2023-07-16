package yabeline

import (
	"bytes"
	"fmt"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func ConvertImage(image []byte) ([]byte, error) {
	resultBuffer := new(bytes.Buffer)
	err := ffmpeg.Input("pipe:", ffmpeg.KwArgs{
		// "f": format,
	}).
		// Filter("scale", ffmpeg.Args{"min'(512,iw)':min'(512,ih)"}).
		Output("pipe:", ffmpeg.KwArgs{
			"c:v":      "png",
			"vf":       "scale=512:-1",
			"frames:v": "1",
			"pix_fmt":  "rgba",
			"f":        "image2",
		}).
		WithInput(bytes.NewBuffer(image)).
		WithOutput(resultBuffer).
		Run()

	if err != nil {
		return nil, err
	}

	resBytes := resultBuffer.Bytes()
	if len(resBytes) == 0 {
		return nil, fmt.Errorf("Result buffer is empty")
	}

	return resBytes, nil
}

func ConvertApng(apng []byte) ([]byte, error) {
	resultBuffer := new(bytes.Buffer)
	err := ffmpeg.Input("-", ffmpeg.KwArgs{
		"f": "apng",
	}).
		Output("-", ffmpeg.KwArgs{
			"framerate": "30",
			"c:v":       "libvpx-vp9",
			"an":        "",
			"vf":        "scale='min(512,iw)':'min(512,ih)':force_original_aspect_ratio=decrease,format=rgba,pad=512:'min(512, ih)':-1:-1:color=0x00000000",
			"f":         "webm",
			"pix_fmt":   "yuva420p",
		}).
		WithInput(bytes.NewBuffer(apng)).
		WithOutput(resultBuffer).
		Run()

	if err != nil {
		return nil, err
	}
	resBytes := resultBuffer.Bytes()
	if len(resBytes) == 0 {
		return nil, fmt.Errorf("Result buffer is empty")
	}
	return resBytes, nil
}
