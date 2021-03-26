package main

import (
	"encoding/json"
	"io/ioutil"
)

type Boundary struct {
	ID          int64       `json:"id"`
	ImgID       int64       `json:"image_id"`
	CategoryID  int64       `json:"category_id"`
	Coordinates [][]float64 `json:"segmentation"`
}

type ImageInfo struct {
	ID       int64  `json:"id"`
	FileName string `json:"file_name"`
}

type AnnotationFile struct {
	Images     []ImageInfo `json:"images"`
	Boundaries []*Boundary `json:"annotations"`
}

func LoadAnnotationFile(path string) (ret *AnnotationFile, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &ret)
	return
}

func BuildFileNameIndex(imgs []ImageInfo) (ret map[int64]string) {
	ret = make(map[int64]string)
	for _, img := range imgs {
		ret[img.ID] = img.FileName
	}

	return
}
