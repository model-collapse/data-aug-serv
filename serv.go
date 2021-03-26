package main

import (
	"log"

	"gocv.io/x/gocv"

	http "github.com/valyala/fasthttp"
)

var scenes map[string]*Scene

func initialize() (err error) {
	if err = LoadConfig("./conf.json"); err != nil {
		return
	}

	if scenes, err = LoadScenes("./scenes.json"); err != nil {
		return
	}

	return
}

func main() {
	if err := initialize(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Initialized")

	handle := func(c *http.RequestCtx) {
		log.Printf("here")
		name := c.URI().QueryArgs().Peek("scene")
		count := c.URI().QueryArgs().GetUintOrZero("count")
		box := c.URI().QueryArgs().Peek("box")

		s, suc := scenes[string(name)]
		if !suc {
			log.Printf("No such scene!")
			return
		}

		img, bboxes, names := s.Generate(scenes[string(name)].randomID(), count)
		if string(box) == "true" {
			drawBoundingBoxOnImage(img, bboxes, names)
		}

		data, err := gocv.IMEncode(gocv.JPEGFileExt, img)

		if err != nil {
			log.Printf("Err [encode] %v", err)
		}

		c.SetContentType("image/jpeg")
		c.Write(data)

		return
	}

	log.Printf("Serving...")
	http.ListenAndServe("0.0.0.0:8093", handle)
}
