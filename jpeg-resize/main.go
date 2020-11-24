package main

import (
	"flag"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
)

func main() {
	flag.Parse()

	var inputFile string
	if args := flag.Args(); len(args) != 1 {
		log.Fatal("Path to input JPEG file must be specified as an arg.")
	} else {
		inputFile = args[0]
	}

	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// decode jpeg into image.Image
	img, err := jpeg.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	// Resize to width using Lanczos resampling and preserve aspect ratio.
	m := resize.Resize(300, 0, img, resize.Bicubic)

	outputFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_resize.jpg"

	out, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// write new image to file
	if err := jpeg.Encode(out, m, nil); err != nil {
		log.Fatal(err)
	}
}
