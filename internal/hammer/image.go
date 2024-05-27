package hammer

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/curtisnewbie/miso/miso"
	"github.com/disintegration/gift"
	_ "golang.org/x/image/webp"
)

func GiftCompressImage(rail miso.Rail, file string, output string) error {
	src, typ, err := loadImage(rail, file)
	if err != nil {
		return fmt.Errorf("failed to load image, filename: %v, %v", file, err)
	}

	imgFilter := gift.New(gift.ResizeToFit(512, 512, gift.LanczosResampling))
	dst := image.NewNRGBA(imgFilter.Bounds(src.Bounds()))
	imgFilter.Draw(dst, src)
	if err := saveImage(rail, output, dst, typ); err != nil {
		return fmt.Errorf("failed to save filtered image, file: %v, %v", output, err)
	}
	return nil
}

func loadImage(rail miso.Rail, filename string) (image.Image, string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open image file, filename: %v, %v", filename, err)
	}
	defer f.Close()
	img, typ, err := image.Decode(f)
	if err != nil {
		rail.Errorf("image decode failed, filename: %v, %v", filename, err)
		return nil, "", err
	}
	return img, typ, nil
}

func saveImage(rail miso.Rail, filename string, img image.Image, typ string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create image file, filename: %v, %v", filename, err)
	}
	defer f.Close()

	rail.Infof("image type: %v, %v", filename, typ)

	switch typ {
	case "png":
		err = png.Encode(f, img)
	case "jpeg":
		err = jpeg.Encode(f, img, nil)
	case "gif":
		err = gif.Encode(f, img, nil)
	default:
		err = png.Encode(f, img)
	}

	if err != nil {
		return fmt.Errorf("image encode failed, filename: %v, %v", filename, err)
	}
	return nil
}
