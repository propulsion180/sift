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

	//"golang.org/x/tools/go/analysis/passes/nilfunc"

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

var imageId string
var largeImage *canvas.Image

func extractExt(filename string) string {
	return strings.ToLower((filepath.Ext(filename)))
}

func isSupportedExt(ext string) bool {
	return slices.Contains(imageExt, ext)
}

//go func() {
//				for i := range images {
//					ext := strings.ToLower(filepath.Ext(images[i].Filepath))
//
//					if ext == ".jpg" || ext == ".jpeg" {
//						thumbPath, err := extractThumbnail(images[i].Filepath)
//						if err == nil {
//							images[i].ThumbPath = thumbPath
//							continue
//						}
//					}
//
//					thumbPath, err := generateThumbnailFromImage(images[i].Filepath, 300)
//					if err == nil {
//						images[i].ThumbPath = thumbPath
//					}
//
//				}
//			}()

func loadFolder(foldep string) ([]ImageData, error) {
	files, err := os.ReadDir(foldep)
	if err != nil {
		return nil, err
	}

	validFiles := 0
	for _, file := range files {
		if !file.IsDir() && isSupportedExt(extractExt(file.Name())) && extractExt(file.Name()) != ".orf" {
			validFiles++
		}
	}

	imgChan := make(chan ImageData, validFiles)

	for _, file := range files {

		if file.IsDir() {
			continue
		}

		ext := extractExt(file.Name())
		if !isSupportedExt(ext) {
			continue
		}

		if ext == ".orf" {
			continue
		}

		fullPath := filepath.Join(foldep, file.Name())
		baseName := strings.TrimSuffix(file.Name(), ext)
		rawName := baseName + ".orf"
		rawPath := filepath.Join(foldep, rawName)

		go func(path string, base string, raw string, extension string) {

			imgData := ImageData{
				Filepath:  path,
				RawPath:   raw,
				ThumbPath: "",
				Format:    ext,
				Rating:    0,
			}

			thumbPath, err := extractThumbnail(fullPath)
			if err == nil {
				imgData.ThumbPath = thumbPath
			} else {
				thumbPath, err := generateThumbnailFromImage(fullPath, 300)
				if err == nil {
					imgData.ThumbPath = thumbPath
				}
			}

			imgChan <- imgData
		}(fullPath, baseName, rawPath, ext)
	}

	var images []ImageData
	for i := 0; i < validFiles; i++ {
		img := <-imgChan
		if img.Filepath != "" {
			images = append(images, img)
		}
	}

	close(imgChan)
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

func setCurrAndSwitchTo(image *ImageData, wdw fyne.Window, layout *fyne.Container) {
	imageId = image.Filepath
	largeImage.File = image.Filepath
	largeImage.Refresh()
	wdw.SetContent(layout)
}

func main() {
	sift := app.New()

	var images []ImageData
	//var imageGrid *fyne.Container

	siftWindow := sift.NewWindow("Sift Main")

	folderLabel := widget.NewLabel("No folder selected")

	largeImage = canvas.NewImageFromFile("")
 	largeImage.FillMode = canvas.ImageFillContain



	var header *fyne.Container

	var fullImageHeader *fyne.Container

	var fullImageLayout *fyne.Container

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

			list := widget.NewList(
				func() int {
					return (len(images) + 3) / 4
				},
				func() fyne.CanvasObject {

					var thumbs []fyne.CanvasObject

					for i := 0; i < 4; i++ {
						img := canvas.NewImageFromFile("")
						img.FillMode = canvas.ImageFillContain
						img.SetMinSize(fyne.NewSize(200, 200))

						btn := widget.NewButton("", nil)
						btn.Importance = widget.LowImportance

						stack := container.NewStack(img, btn)
						thumbs = append(thumbs, stack)
					}

					return container.NewGridWithColumns(4, thumbs...)
				},
				func(id widget.ListItemID, obj fyne.CanvasObject) {
					row := obj.(*fyne.Container)
					startIndex := id * 4

					for i := 0; i < 4; i++ {
						idx := startIndex + i
						stackCont := row.Objects[i].(*fyne.Container)
						img := stackCont.Objects[0].(*canvas.Image)
						btn := stackCont.Objects[1].(*widget.Button)

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

							btn.OnTapped = func() { setCurrAndSwitchTo(&imaged, siftWindow, fullImageLayout) }

						} else {
							img.File = ""
							img.Refresh()
						}
					}
				},
			)

			mainlayout := container.NewBorder(
				header,
				nil,
				nil,
				nil,
				list,
			)
			siftWindow.SetContent(mainlayout)
		}, siftWindow)
	})

	fullPageTestButton := widget.NewButton("Full Page", nil)
	returnToGrid := widget.NewButton("< Back", nil)

	header = container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			fullPageTestButton,
			selectFolderBtn,
			folderLabel,
		),
		widget.NewSeparator(),
	)

	fullImageHeader = container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			returnToGrid,
			folderLabel,
		),
		widget.NewSeparator(),
	)

	//header.Objects[0] = container.NewBorder(
	//	nil,
	//	nil,
	//	fullPageTestButton,
	//	selectFolderBtn,
	//	folderLabel,
	//)
	header.Refresh()

	mainLayout := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		nil,
	)

	fullImageFooter := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabel("test footer"),
	)

	fullImageLayout = container.NewBorder(
		fullImageHeader,
		fullImageFooter,
		nil,
		nil,
		largeImage,
	)

	fullPageTestButton.OnTapped = func() {
		siftWindow.SetContent(fullImageLayout)
	}

	returnToGrid.OnTapped = func() {
		imageId = ""
		largeImage.File = ""
		largeImage.Refresh()
		siftWindow.SetContent(mainLayout)
	}

	siftWindow.SetContent(mainLayout)
	siftWindow.Resize(fyne.NewSize(800, 600))
	siftWindow.ShowAndRun()
}
