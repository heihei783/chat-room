package upload

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

func ResizeImage(srcPath, dstPath string, width, height uint) error {
	// 打开源文件
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 解码图片
	var img image.Image
	ext := filepath.Ext(srcPath)
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	default:
		return fmt.Errorf("不支持的图片格式: %s", ext)
	}
	if err != nil {
		return err
	}

	// 调整大小
	resized := resize.Resize(width, height, img, resize.Lanczos3)

	// 保存调整后的图片
	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(out, resized, nil)
	case ".png":
		return png.Encode(out, resized)
	}

	return nil
}

func GenerateThumbnail(srcPath, dstPath string) error {
	return ResizeImage(srcPath, dstPath, 200, 200)
}
