package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
)

type ImageData struct {
	Filepath string
	RawPath  string
	Format   string
	Rating   int
	img      *canvas.Image
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

func extractNum(fp string) (int, error) {
	filename := filepath.Base(fp)
	minusExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	numberString := strings.TrimPrefix(minusExt, "PC")
	return strconv.Atoi(numberString)
}

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

			imageLoaded := canvas.NewImageFromFile(path)

			imgData := ImageData{
				Filepath: path,
				RawPath:  raw,
				Format:   ext,
				img:      imageLoaded,
				Rating:   0,
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

	slices.SortFunc(images, func(a, b ImageData) int {
		aNum, err := extractNum(a.Filepath)
		if err != nil {
			return 0
		}
		bNum, err := extractNum(b.Filepath)
		if err != nil {
			return 0
		}

		if aNum > bNum {
			return -1
		}
		if aNum < bNum {
			return 1
		}
		return 0
	})

	return images, nil
}

func main() {
	sift := app.New()

	var images []ImageData
	//var imageGrid *fyne.Container
	siftWindow := sift.NewWindow("Sift Main")

	folderLabel := widget.NewLabel("No folder selected")

	currIndex := 0

	var header *fyne.Container
	var footer *fyne.Container
	var mainLayout *fyne.Container

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
			imagesList, err := loadFolder(uri.Path())
			if err != nil {
				dialog.ShowError(err, siftWindow)
				return
			}

			images = imagesList
			mainLayout.Objects = append(mainLayout.Objects, images[currIndex].img)
			mainLayout.Refresh()

		}, siftWindow)
	})

	previousButton := widget.NewButton("<-", func() {
		if currIndex <= 0 {
			return
		}
		currIndex--
		mainLayout.Objects[2] = images[currIndex].img
		mainLayout.Refresh()
	})
	nextButton := widget.NewButton("->", func() {
		if currIndex >= len(images) {
			return
		}
		currIndex++
		mainLayout.Objects[2] = images[currIndex].img
		mainLayout.Refresh()
	})

	footer = container.NewHBox(
		previousButton,
		widget.NewSeparator(),
		nextButton,
	)

	header = container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			nil,
			selectFolderBtn,
			folderLabel,
		),
		widget.NewSeparator(),
	)

	header.Refresh()

	mainLayout = container.NewBorder(
		header,
		footer,
		nil,
		nil,
		nil,
	)

	siftWindow.SetContent(mainLayout)
	siftWindow.Resize(fyne.NewSize(800, 600))
	siftWindow.ShowAndRun()
}
