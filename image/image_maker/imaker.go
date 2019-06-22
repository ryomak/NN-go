package imaker

import (
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/muesli/smartcrop"
	"github.com/muesli/smartcrop/nfnt"
	"github.com/nfnt/resize"
	"github.com/ryomak/deep-learning-go/util"
)

type ImageMakerUtil struct {
	LearnDir    string
	AnswerDir   string
	OutputFile  string
	ImageHeight int
	ImageWidth  int
}

func Init(learnDir, answerDir, outputFile string, imageWidth, imageHeight int) ImageMakerUtil {
	return ImageMakerUtil{
		LearnDir:    learnDir,
		AnswerDir:   answerDir,
		OutputFile:  outputFile,
		ImageHeight: imageHeight,
		ImageWidth:  imageWidth,
	}
}

func (i ImageMakerUtil) Decode(path string) ([]float64, error) {
	cDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Join(cDir, path))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	src, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	src = i.cropping(src)
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < h {
		w = h
	} else {
		h = w
	}
	binData := make([]float64, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			binData[y*w*3+x*3] = float64(r>>8) / 255.0
			binData[y*w*3+x*3+1] = float64(g>>8) / 255.0
			binData[y*w*3+x*3+2] = float64(b>>8) / 255.0
		}
	}
	return binData, nil
}

func (iu ImageMakerUtil) Encode(b []float64) (interface{}, error) {
	img := image.NewRGBA(image.Rect(0, 0, iu.ImageWidth, iu.ImageHeight))
	for i := 0; i < len(b); i = i + 3 {
		img.Set(i/3%iu.ImageWidth, (i/3)/iu.ImageHeight, color.RGBA{uint8(b[i] * 255), uint8(b[i+1] * 255), uint8(b[i+2] * 255), 255})
	}
	f, err := util.OpenOrCreateFile(iu.OutputFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return nil, jpeg.Encode(f, img, nil)
}

func (i ImageMakerUtil) MakePattern() ([]util.DataSet, error) {
	patterns := []util.DataSet{}
	learnSet, err := i.loadDataSet(i.LearnDir)
	if err != nil {
		return nil, err
	}
	answerSet, err := i.loadDataSet(i.AnswerDir)
	if err != nil {
		return nil, err
	}
	for key, learn := range learnSet {
		if answerSet[key] == nil {
			log.Println("can't find :", key)
			continue
		}
		patterns = append(patterns, util.DataSet{Input: learn, Response: answerSet[key]})
	}
	return patterns, nil
}

func (i ImageMakerUtil) loadDataSet(dirPath string) (map[string][]float64, error) {
	result := map[string][]float64{}
	names, err := util.OpenDirFiles(dirPath)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		fname := filepath.Join(dirPath, name)
		ff, err := i.Decode(fname)
		if err != nil {
			log.Println(fname, " can't decode")
			continue
		}
		result[name] = ff
	}
	return result, nil
}

func (i ImageMakerUtil) cropping(img image.Image) image.Image {
	analyzer := smartcrop.NewAnalyzer(nfnt.NewDefaultResizer())
	topCrop, err := analyzer.FindBestCrop(img, i.ImageHeight, i.ImageWidth)
	if err == nil {
		type SubImager interface {
			SubImage(r image.Rectangle) image.Image
		}
		img = img.(SubImager).SubImage(topCrop)
	}
	return resize.Resize(uint(i.ImageHeight), uint(i.ImageWidth), img, resize.Lanczos3)
}
