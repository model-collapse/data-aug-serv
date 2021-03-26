package main

import (
	"encoding/json"
	"image"
	"io/ioutil"
	"log"
	"math"
	"math/rand"

	"gocv.io/x/gocv"
)

type SampleParameter struct {
	Mode  string  `json:"mode"`
	Delta float32 `json:"delta"`
	//X     float32 `json:"x"`
	//Y float32 `json:"y"`
}

type SizeParamter struct {
	Mode     string  `json:"mode"`
	Max      int     `json:"max"`
	Min      int     `json:"min"`
	LocSigma float32 `json:"loc_sigma"`
}

type AffineParameter struct {
	MaxShear float32 `json:"max_shear"`
	MaxComp  float32 `json:"max_comp"`
	Rotation bool    `json:"rotation"`
	Flip     bool    `json:"flip"`
}

type ColorParameter struct {
	MaxBrightnessDelta float32 `json:"max_brightness_delta"`
	MaxContrastDelta   float32 `json:"max_contrast_delta"`
	MaxHueShiftDelta   float32 `json:"max_hue_shift_delta"`
}

type CropParameter struct {
	MaxHeightReduce float32 `json:"max_height_reduce"`
	MaxWidthReduce  float32 `json:"max_width_reduce"`
}

type LensParameter struct {
}

type ObjectParameters struct {
	Size   SizeParamter    `json:"size_param"`
	Affine AffineParameter `json:"affine_param"`
	Color  ColorParameter  `json:"color_param"`
}

type BackgroundParameters struct {
	Crop  CropParameter  `json:"crop_param"`
	Color ColorParameter `json:"color_param"`
}

type Scene struct {
	ImagePaths  []string             `json:"image_paths"`
	ObjectPaths []string             `json:"object_paths"`
	Object      ObjectParameters     `json:"obj_param"`
	Background  BackgroundParameters `json:"back_param"`
	//Sample      SampleParameter      `json:"sample_param"`
}

func LoadScenes(path string) (ret map[string]*Scene, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	if err = json.Unmarshal(data, &ret); err != nil {
		return
	}

	for _, s := range ret {
		if s.ImagePaths[0] == "all" {
			fsl, _ := ioutil.ReadDir(GConf.BackgroundDir)
			log.Printf("#background = %d", len(fsl))
			s.ImagePaths = make([]string, 0, len(fsl))
			for _, f := range fsl {
				s.ImagePaths = append(s.ImagePaths, f.Name())
			}
		}

		if s.ObjectPaths[0] == "all" {
			fsl, _ := ioutil.ReadDir(GConf.ObjectDir)
			log.Printf("#object = %d", len(fsl))
			s.ObjectPaths = make([]string, 0, len(fsl))
			for _, f := range fsl {
				s.ObjectPaths = append(s.ObjectPaths, f.Name())
			}
		}
	}

	return
}

func adjustBrightnessAndContrast(img *gocv.Mat, color ColorParameter) {

}

func (s *Scene) randomID() int {
	return rand.Intn(len(s.ImagePaths))
}

func (s *Scene) generateBackground(imgID int) (r gocv.Mat, dx, dy int) {
	path := GConf.BackgroundDir + "/" + s.ImagePaths[imgID]
	baseImg := gocv.IMRead(path, gocv.IMReadColor)
	log.Printf("background format = %v", baseImg.Type())

	mhr := int(float32(baseImg.Rows()) * s.Background.Crop.MaxHeightReduce)
	mwr := int(float32(baseImg.Cols()) * s.Background.Crop.MaxWidthReduce)
	hr := rand.Intn(mhr)
	wr := rand.Intn(mwr)

	dy = rand.Intn(hr)
	dx = rand.Intn(wr)

	r = baseImg.Region(image.Rectangle{Min: image.Point{dx, dy}, Max: image.Point{dx + baseImg.Cols() - wr, dy + baseImg.Rows() - hr}})
	r = r.Clone()
	adjustBrightnessAndContrast(&r, s.Background.Color)

	return
}

func (s *Scene) generateAnObject(size int) (r gocv.Mat, name string) {
	name = s.ObjectPaths[rand.Intn(len(s.ObjectPaths))]
	path := GConf.ObjectDir + "/" + name
	objImg := gocv.IMRead(path, gocv.IMReadUnchanged)

	log.Printf("object format = %v", objImg.Type())
	aspectRatioHW := float32(objImg.Rows()) / float32(objImg.Cols())
	cmpr := rand.Float32() * s.Object.Affine.MaxComp

	if rand.Intn(1) == 0 {
		aspectRatioHW += cmpr
	} else {
		aspectRatioHW -= cmpr
	}

	var nh int
	var nw int
	if aspectRatioHW > 1 {
		nh = size
		nw = int(float32(size) / aspectRatioHW)
	} else {
		nw = size
		nh = int(float32(size) * aspectRatioHW)
	}

	objImgRes := gocv.NewMat()
	gocv.Resize(objImg, &objImgRes, image.Point{nw, nh}, 0, 0, gocv.InterpolationLinear)

	diag := arrToMtx([][]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}})
	lastRow := arrToMtx([][]float64{{0, 0, 1}})
	// Start affine
	// Rotation
	var rot gocv.Mat
	if s.Object.Affine.Rotation {
		angle := rand.Float32() * 2 * float32(math.Pi)
		rot = gocv.GetRotationMatrix2D(image.Point{nw / 2, nh / 2}, float64(angle), 1)
		gocv.Vconcat(rot, lastRow, &rot)
	} else {
		rot = diag
	}
	defer rot.Close()

	//Shearing
	shear := rand.Float64() * float64(s.Object.Affine.MaxShear)
	shearTranslate := shear * float64(nh) / 2
	shearMtxArr := [][]float64{{1, shear, -shearTranslate}, {0, 1, 0}, {0, 0, 1}}
	shearMtx := arrToMtx(shearMtxArr)
	defer shearMtx.Close()

	nnw := shear*float64(nh) + float64(nw)
	nnsize := int(math.Sqrt(nnw*nnw + float64(nh*nh)))

	var flipMtx gocv.Mat
	// Flip
	if rand.Intn(1) == 0 && s.Object.Affine.Flip {
		flipMtx = arrToMtx([][]float64{{-1, 0, float64(nnsize)}, {0, 1, 0}, {0, 0, 1}})
	} else {
		flipMtx = diag
	}

	transM := rot.MultiplyMatrix(shearMtx)
	log.Printf("shape of trans = (%d, %d)", transM.Cols(), transM.Rows())
	log.Printf("shape of rot = (%d, %d)", rot.Cols(), rot.Rows())
	log.Printf("type of trans = %v, type of rot = %v", transM.Type(), rot.Type())
	transM = flipMtx.MultiplyMatrix(transM)
	transM = transM.Region(image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{3, 2}})
	defer transM.Close()

	r = gocv.NewMat()
	//apply
	gocv.WarpAffine(objImgRes, &r, transM, image.Point{nnsize, nnsize})
	r = shrinkToBoundingBox(r)

	adjustBrightnessAndContrast(&r, s.Object.Color)
	return
}

func extractAlpha(img gocv.Mat) (r gocv.Mat) {
	r = gocv.NewMatWithSize(img.Rows(), img.Cols(), gocv.MatTypeCV8UC1)
	for y := 0; y < img.Rows(); y++ {
		for x := 0; x < img.Cols(); x++ {
			r.SetUCharAt(y, x, img.GetUCharAt(y, x*4+3))
		}
	}

	return
}

func shrinkToBoundingBox(img gocv.Mat) gocv.Mat {
	colSum := gocv.NewMat()
	defer colSum.Close()
	rowSum := gocv.NewMat()
	defer rowSum.Close()

	imgAlpha := extractAlpha(img)
	defer imgAlpha.Close()

	gocv.Reduce(imgAlpha, &colSum, 0, gocv.ReduceSum, gocv.MatTypeCV32F)
	gocv.Reduce(imgAlpha, &rowSum, 1, gocv.ReduceSum, gocv.MatTypeCV32F)

	//log.Printf("colSize size = %d, sum = %f", colSum.Cols(), colSum.Sum().Val1)

	var bbox image.Rectangle
	for i := 0; i < colSum.Cols(); i++ {
		if colSum.GetFloatAt(0, i) > 1.0 {
			bbox.Min.X = i
			break
		}
	}

	for i := colSum.Cols() - 1; i >= 0; i-- {
		if colSum.GetFloatAt(0, i) > 1.0 {
			bbox.Max.X = i + 1
			break
		}
	}

	for i := 0; i < rowSum.Rows(); i++ {
		if rowSum.GetFloatAt(i, 0) > 1.0 {
			bbox.Min.Y = i
			break
		}
	}

	for i := rowSum.Rows() - 1; i >= 0; i-- {
		if rowSum.GetFloatAt(i, 0) > 1.0 {
			bbox.Max.Y = i + 1
			break
		}
	}

	log.Printf("bbox = %v", bbox)
	return img.Region(bbox)
}

func arrToMtx(arr [][]float64) (r gocv.Mat) {
	r = gocv.NewMatWithSize(len(arr), len(arr[0]), gocv.MatTypeCV64FC1)
	for y := 0; y < r.Rows(); y++ {
		for x := 0; x < r.Cols(); x++ {
			r.SetDoubleAt(y, x, arr[y][x])
		}
	}

	return
}

func (s *Scene) sampleALocation(w, h int, dx, dy int) image.Point {
	xx := rand.NormFloat64() * float64(w) / 2 / 2.5
	yy := math.Abs(rand.NormFloat64() * float64(h) / 2.5)

	x := float32(w/2) + float32(xx) - float32(dx)
	y := float32(h) - float32(yy) - float32(dy)

	if x < 0 {
		x = 0
	}

	if y < 0 {
		y = 0
	}

	if x >= float32(w) {
		x = float32(w - 2)
	}

	if y >= float32(h) {
		y = float32(h - 2)
	}

	return image.Point{int(x), int(y)}
}

func (s *Scene) estimateSizeRatio(x, y int, w, h int) float32 {
	if s.Object.Size.Mode == "distance" {
		return s.Object.Size.LocSigma + (1-s.Object.Size.LocSigma)*(float32(y)/float32(h))
	}

	return 1.0
}

func (s *Scene) render(bk, fore gocv.Mat, x, y int) error {
	bw, bh := bk.Cols(), bk.Rows()
	fw, fh := fore.Cols(), fore.Rows()

	bsx, bsy := x-fw/2, y-fh/2
	bex, bey := bsx+fw, bsy+fh

	fsx, fsy := 0, 0
	fex, fey := fw, fh

	if bsx < 0 {
		fsx = -bsx
		bsx = 0
	}

	if bsy < 0 {
		fsy = -bsy
		bsy = 0
	}

	if bex >= bw {
		fex = fw - (bex - bw)
		bex = bw
	}

	if bey >= bh {
		fey = fh - (bey - bh)
		bey = bh
	}

	log.Printf("bw,bh = %d,%d", bw, bh)
	log.Printf("bex,bey = %d,%d", bex, bey)

	fpatch := fore.Region(image.Rectangle{Min: image.Point{fsx, fsy}, Max: image.Point{fex, fey}})
	for y := 0; y < fpatch.Rows(); y++ {
		for x := 0; x < fpatch.Cols(); x++ {
			alpha := fpatch.GetUCharAt(y, x*4+3)
			alphaV := float32(alpha) / 255.0
			if alphaV <= 0 {
				continue
			}
			for c := 0; c < 3; c++ {
				val := float32(fpatch.GetUCharAt(y, x*4+c))*alphaV + float32(bk.GetUCharAt(y+bsy, (x+bsx)*3+c))*(1-alphaV)
				bk.SetUCharAt(y+bsy, (x+bsx)*3+c, uint8(int(val)))
			}
		}
	}

	return nil
}

func (s *Scene) Generate(imgID int, n int) (r gocv.Mat, bboxes []image.Rectangle, names []string) {
	bk, dx, dy := s.generateBackground(imgID)
	log.Printf("image [%d], size=(%d,%d), delta=(%d,%d)", imgID, bk.Cols(), bk.Rows(), dx, dy)
	for i := 0; i < n; i++ {
		p := s.sampleALocation(bk.Cols(), bk.Rows(), dx, dy)
		sizeRatio := s.estimateSizeRatio(p.X, p.Y, bk.Cols(), bk.Rows())

		size := s.Object.Size.Min + rand.Intn(s.Object.Size.Max-s.Object.Size.Min)
		size = int(float32(size) * sizeRatio)

		obj, n := s.generateAnObject(size)
		log.Printf("rendering on (%d,%d) with size = (%d,%d)...", p.X, p.Y, obj.Cols(), obj.Rows())
		if err := s.render(bk, obj, p.X, p.Y); err != nil {
			log.Printf("Err [render] %v", err)
		}

		bbox := image.Rectangle{
			Min: image.Point{X: p.X - obj.Cols()/2, Y: p.Y - obj.Rows()/2},
		}
		bbox.Max = image.Point{X: bbox.Min.X + obj.Cols(), Y: bbox.Min.Y + obj.Rows()}

		bboxes = append(bboxes, bbox)
		names = append(names, n)
	}

	r = bk
	return
}

func ListObjects(path string) (ret []string, err error) {
	lst, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}

	ret = make([]string, 0, len(lst))
	for _, f := range lst {
		ret = append(ret, f.Name())
	}

	return
}
