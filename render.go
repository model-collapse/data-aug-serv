package main

import (
	"image"
	"image/color"
	"log"

	"gocv.io/x/gocv"
)

func drawBoundingBoxOnImage(img gocv.Mat, bboxes []image.Rectangle, names []string) {
	for i, bbox := range bboxes {
		log.Printf("rendering... %v", bbox)
		gocv.Rectangle(&img, bbox, color.RGBA{255, 255, 0, 0}, 1)
		gocv.PutText(&img, names[i], bbox.Max, gocv.FontHersheyComplex, 0.5, color.RGBA{255, 0, 0, 255}, 1)
	}
}
