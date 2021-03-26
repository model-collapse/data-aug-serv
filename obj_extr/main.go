package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"runtime/debug"
	"sync"

	"github.com/llgcode/draw2d/draw2dimg"
)

func MinInt(a, b int) int {
	if a > b {
		return b
	}

	return a
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func extractBoundingBox(bds []float64) (r image.Rectangle) {
	r.Min = image.Point{X: math.MaxInt32, Y: math.MaxInt32}

	for i := 0; i < len(bds); i += 2 {
		x := bds[i]
		y := bds[i+1]

		r.Min.X = MinInt(r.Min.X, int(x))
		r.Min.Y = MinInt(r.Min.Y, int(y))

		r.Max.X = MaxInt(r.Max.X, int(x))
		r.Max.Y = MaxInt(r.Max.Y, int(y))
	}

	return
}

func extractObject(fns map[int64]string, b *Boundary) error {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("Panic = %v, stack = %s", e, debug.Stack())
		}
	}()

	bc := b.Coordinates[0]
	bnd := extractBoundingBox(bc)

	fn, ok := fns[b.ImgID]
	if !ok {
		return fmt.Errorf("image id %d, does not exist", b.ImgID)
	}

	f, err := os.Open("../../TACO/data/" + fn)
	if err != nil {
		return err
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		return err
	}

	ibound := img.Bounds()
	if bnd.Max.X > ibound.Max.X ||
		bnd.Max.Y > ibound.Max.Y {
		return fmt.Errorf("boundary out of image scope")
	}

	nbnd := image.Rectangle{Max: image.Point{bnd.Max.X - bnd.Min.X, bnd.Max.Y - bnd.Min.Y}}

	patch := image.NewRGBA(nbnd)
	for y := 0; y < patch.Rect.Max.Y; y++ {
		for x := 0; x < patch.Rect.Max.X; x++ {
			cc := img.At(x+bnd.Min.X, y+bnd.Min.Y)
			patch.Set(x, y, cc)
		}
	}

	mask := image.NewRGBA(nbnd)
	gc := draw2dimg.NewGraphicContext(mask)
	gc.SetFillColor(color.RGBA{0, 0, 0, 255})

	gc.MoveTo(bc[len(bc)-2]-float64(bnd.Min.X), bc[len(bc)-1]-float64(bnd.Min.Y))
	for i := 0; i < len(bc); i += 2 {
		gc.LineTo(bc[i]-float64(bnd.Min.X), bc[i+1]-float64(bnd.Min.Y))
	}

	gc.Close()
	gc.FillStroke()

	for y := 0; y < patch.Rect.Max.Y; y++ {
		for x := 0; x < patch.Rect.Max.X; x++ {
			r, g, b, _ := patch.At(x, y).RGBA()
			_, _, _, a := mask.At(x, y).RGBA()
			c := color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
			patch.SetRGBA(x, y, c)
		}
	}

	fn = fmt.Sprintf("objs/%d.png", b.ID)
	fw, err := os.OpenFile(fn, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer fw.Close()

	if err := png.Encode(fw, patch); err != nil {
		return err
	}

	return nil
}

func main() {
	annFile, err := LoadAnnotationFile("../../TACO/data/annotations.json")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("#boundaries = %d", len(annFile.Boundaries))
	log.Printf("#images = %d", len(annFile.Images))

	fns := BuildFileNameIndex(annFile.Images)

	chBnd := make(chan *Boundary, 100)
	go func() {
		for _, b := range annFile.Boundaries {
			chBnd <- b
		}

		close(chBnd)
	}()

	wg := sync.WaitGroup{}
	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			for b := range chBnd {
				//log.Printf("here")
				if err := extractObject(fns, b); err != nil {
					log.Print(err)
				}
			}

			wg.Done()
		}()
	}

	wg.Wait()
}
