package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	//	"golang.org/x/tools/go/analysis/passes/defers"
	//
	// "github.com/fredbi/uri"
	"github.com/rwcarlsen/goexif/exif"
)

type ImageData struct {
	Filepath  string
	RawPath   string
	ThumbPath string
	Format    string
	Rating    int
}

var imageExt = []string{
	".jpg",
	".jpeg",
	".png",
	".orf",
}

func extractExt(filename string) string {
	return strings.ToLower((filepath.Ext(filename)))
}

func isSupportedExt(ext string) bool {
	return slices.Contains(imageExt, ext)
}

func loadFolder(foldep string) ([]ImageData, error) {
	imagesmap := make(map[string]*ImageData)

	files, err := os.ReadDir(foldep)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := extractExt(file.Name())
		if !isSupportedExt(ext) {
			continue
		}

		fullPath := filepath.Join(foldep, file.Name())
		baseName := strings.TrimSuffix(file.Name(), ext)

		if imagesmap[baseName] == nil {
			imagesmap[baseName] = &ImageData{"", "", "", ext, 0}
		}

		if ext == ".jpg" || ext == ".jpeg" {
			imagesmap[baseName].Filepath = fullPath
		} else if ext == ".orf" {
			imagesmap[baseName].RawPath = fullPath
		}
	}

	var images []ImageData
	for _, im := range imagesmap {
		if im.Filepath != "" {
			images = append(images, *im)
		}
	}

	return images, nil
}

func createThumb(imagePath string) string {
	fmt.Println(imagePath)

	cacheDir := filepath.Join(os.TempDir(), "sift-thumbs")
	os.Mkdir(cacheDir, 0755)

	hash := md5.Sum([]byte(imagePath))
	return filepath.Join(cacheDir, hex.EncodeToString(hash[:])+".jpg")
}

func extractThumbnail(imagePath string) (string, error) {
	thumbPath := createThumb(imagePath)

	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return "", fmt.Errorf("no EXIF data: %w", err)
	}

	thumbImg, err := x.JpegThumbnail()
	if err != nil {
		return "", fmt.Errorf("no embedded thumbnail: %w", err)
	}

	out, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = out.Write(thumbImg)
	if err != nil {
		return "", err
	}

	return thumbPath, nil
}

func generateThumbnailFromImage(srcPath string, maxSize int) (string, error) {
	thumbPath := createThumb(srcPath)

	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, nil
	}

	file, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var newWidth, newHeight int
	if width > height {
		newWidth = maxSize
		newHeight = height * maxSize / width
	} else {
		newHeight = maxSize
		newWidth = width * maxSize / height
	}

	thumb := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := x * width / newWidth
			srcY := y * height / newHeight
			thumb.Set(x, y, img.At(srcX, srcY))
		}
	}

	out, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	err = jpeg.Encode(out, thumb, &jpeg.Options{Quality: 85})
	return thumbPath, err

}

func generateThumbnail(srcPath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(srcPath))

	if ext == ".jpg" || ext == ".jpeg" {
		thumbPath, err := extractThumbnail(srcPath)
		if err == nil {
			return thumbPath, nil
		}
	}

	return generateThumbnailFromImage(srcPath, 300)
}

func main() {
	sift := app.New()

	var images []ImageData
	//var imageGrid *fyne.Container

	siftWindow := sift.NewWindow("Sift Main")

	folderLabel := widget.NewLabel("No folder selected")

	selectFolderBtn := widget.NewButton("Select folder", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, siftWindow)
				return
			}
			if uri == nil {
				return
			}
			folderLabel.SetText("Folder: " + uri.Path())
			images, err = loadFolder(uri.Path())
			if err != nil {
				dialog.ShowError(err, siftWindow)
				return
			}

			//			var thumbnails []fyne.CanvasObject
			//			for _, img := range images {
			//				thumb := createThumb(img.Filepath)
			//				thumbnails = append(thumbnails, thumb)
			//			}
			//
			//			imageGrid = container.NewGridWithColumns(4, thumbnails...)
			//			scrollContent := container.NewScroll(imageGrid)

			go func() {
				for i := range images {
					ext := strings.ToLower(filepath.Ext(images[i].Filepath))

					if ext == ".jpg" || ext == ".jpeg" {
						thumbPath, err := extractThumbnail(images[i].Filepath)
						if err == nil {
							images[i].ThumbPath = thumbPath
							continue
						}
					}

					thumbPath, err := generateThumbnailFromImage(images[i].Filepath, 300)
					if err == nil {
						images[i].ThumbPath = thumbPath
					}

				}
			}()

			list := widget.NewList(
				func() int {
					return (len(images) + 3) / 4
				},
				func() fyne.CanvasObject {
					img1 := canvas.NewImageFromFile("")
					img1.FillMode = canvas.ImageFillContain
					img1.SetMinSize(fyne.NewSize(200, 200))
					img2 := canvas.NewImageFromFile("")
					img2.FillMode = canvas.ImageFillContain
					img2.SetMinSize(fyne.NewSize(200, 200))
					img3 := canvas.NewImageFromFile("")
					img3.FillMode = canvas.ImageFillContain
					img3.SetMinSize(fyne.NewSize(200, 200))
					img4 := canvas.NewImageFromFile("")
					img4.FillMode = canvas.ImageFillContain
					img4.SetMinSize(fyne.NewSize(200, 200))

					return container.NewGridWithColumns(4, img1, img2, img3, img4)
				},
				func(id widget.ListItemID, obj fyne.CanvasObject) {
					row := obj.(*fyne.Container)
					startIndex := id * 4

					for i := 0; i < 4; i++ {
						idx := startIndex + i
						img := row.Objects[i].(*canvas.Image)

						if idx < len(images) {
							imaged := images[idx]

							if imaged.ThumbPath != "" {
								img.File = imaged.ThumbPath
								img.Refresh()
							} else {
								img.File = ""
								img.Refresh()

								go func(imgData ImageData, imgWidget *canvas.Image) {
									thumbPath, err := generateThumbnail(imgData.Filepath)
									if err == nil {
										imgWidget.File = thumbPath
										imgWidget.Refresh()
									}
								}(imaged, img)
							}

						} else {
							img.File = ""
							img.Refresh()
						}
					}
				},
			)
			mainlayout := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				list,
			)
			siftWindow.SetContent(mainlayout)
		}, siftWindow)
	})

	header := container.NewBorder(
		nil, nil,
		nil,
		selectFolderBtn,
		folderLabel,
	)

	mainLayout := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		nil,
	)

	siftWindow.SetContent(mainLayout)
	siftWindow.Resize(fyne.NewSize(800, 600))
	siftWindow.ShowAndRun()
}
