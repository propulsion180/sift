package main

import (
	"fmt"
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
	// "github.com/fredbi/uri"
)

type ImageData struct {
	Filepath string
	RawPath  string
	Format   string
	Rating   int
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
			imagesmap[baseName] = &ImageData{"", "", ext, 0}
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

func createThumb(imagePath string) *canvas.Image {
	fmt.Println(imagePath)
	img := canvas.NewImageFromFile(imagePath)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(150, 150))
	return img
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

			//			var thumbnails []fyne.CanvasObject
			//			for _, img := range images {
			//				thumb := createThumb(img.Filepath)
			//				thumbnails = append(thumbnails, thumb)
			//			}
			//
			//			imageGrid = container.NewGridWithColumns(4, thumbnails...)
			//			scrollContent := container.NewScroll(imageGrid)
			imageCache := make(map[int]*canvas.Image)
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
							if cached, ok := imageCache[idx]; ok {
								img.File = cached.File
								img.Refresh()
							} else {
								img.File = ""
								img.Refresh()

								imagePath := images[idx].Filepath
								imageIdx := idx
								go func() {
									loadedImg := canvas.NewImageFromFile(imagePath)
									loadedImg.FillMode = canvas.ImageFillContain
									imageCache[imageIdx] = loadedImg

									img.File = imagePath
									img.Refresh()
								}()
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
