package scripts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/epchao/millionaire-tracker/database"
	"github.com/epchao/millionaire-tracker/models"
	"github.com/kkdai/youtube/v2"
	_ "github.com/lib/pq"
	"github.com/otiai10/gosseract/v2"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"gocv.io/x/gocv"
	"gorm.io/gorm"
)

var (
	revenueRegex  = regexp.MustCompile(`\+\s\$(\d+)\s\(revenue\)`)
	expensesRegex = regexp.MustCompile(`\-\s\$(\d+)\s\(expenses\)`)
	titleRegex    = regexp.MustCompile(`Day\s(\d+)\s#millionaireinthemaking`)
)

type Message struct {
	Items []Item
}

type Item struct {
	Shorts        []Short
	NextPageToken string
}

type Short struct {
	VideoID string
	Title   string
}

// //////////////////
//	DB OPERATIONS  //
// //////////////////

func Initalize() {
	channelId := "UC1htp5BzPQ6ScCL6VpepuvA"
	apiUrl := "https://yt0.lemnoslife.com/channels?part=shorts&id=" + channelId
	shorts, pageToken, err := getShorts(apiUrl)
	throwError(err)
	for _, short := range shorts {
		err = insertShort(short)
		throwError(err)
	}

	for len(pageToken) > 0 {
		apiUrl = "https://yt0.lemnoslife.com/channels?part=shorts&id=" + channelId + "&pageToken=" + pageToken
		newShorts, newPageToken, err := getShorts(apiUrl)
		throwError(err)
		for _, short := range newShorts {
			err = insertShort(short)
			throwError(err)
		}
		pageToken = newPageToken
	}
}

func Update() {
	channelId := "UC1htp5BzPQ6ScCL6VpepuvA"
	apiUrl := "https://yt0.lemnoslife.com/channels?part=shorts&id=" + channelId
	shorts, _, err := getShorts(apiUrl)
	throwError(err)
	for _, short := range shorts {
		result := isShortInDB(short)
		throwError(err)
		if result {
			break // shorts from now on are already registered
		} else {
			err = insertShort(short)
			throwError(err)
		}
	}
}

///////////////////
//  PARSE VIDEO  //
///////////////////

func extractIncome(videoID string) (text string, err error) {
	URL := getVideoData(videoID)
	imagePath, err := downloadLastFrame(videoID, URL)
	throwError(err)
	preProcessImage(imagePath)
	text, err = applyOCR(imagePath)
	throwError(err)
	return text, nil
}

func getVideoData(videoID string) (URL string) {
	fmt.Println("Downloading:", videoID)
	yt := youtube.Client{}
	video, err := yt.GetVideoContext(context.Background(), videoID)
	throwError(err)
	format := video.Formats[0] // best quality video is always first (easiest for us to parse)
	return format.URL
}

func downloadLastFrame(videoID string, URL string) (imagePath string, err error) {
	imagePath = "./out/" + videoID + ".png"
	err = ffmpeg.Input(URL).
		Filter("reverse", ffmpeg.Args{}).
		Output(imagePath, ffmpeg.KwArgs{"frames:v": 1}).
		OverWriteOutput().
		ErrorToStdOut().
		Run()
	throwError(err)
	return imagePath, err
}

func preProcessImage(imagePath string) {
	img := gocv.IMRead(imagePath, gocv.IMReadGrayScale)
	defer img.Close()

	rm := img.Clone()
	defer rm.Close()

	white := gocv.NewMatWithSize(rm.Rows(), rm.Cols(), rm.Type())
	defer white.Close()
	scalar := gocv.NewScalar(255.0, 255.0, 255.0, 255.0)
	white.SetTo(scalar)
	gocv.Subtract(white, rm, &rm)

	// biniarization
	threshImg := gocv.NewMat()
	defer threshImg.Close()
	gocv.Threshold(img, &threshImg, 0, 255, gocv.ThresholdOtsu+gocv.ThresholdBinary)

	// skeleton
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Pt(2, 2))
	defer kernel.Close()

	// thinning
	dilateImg := gocv.NewMat()
	defer dilateImg.Close()
	gocv.MorphologyExWithParams(rm, &dilateImg, gocv.MorphErode, kernel, 1, gocv.BorderConstant)

	preResult := gocv.NewMat()
	defer preResult.Close()
	gocv.CvtColor(threshImg, &threshImg, gocv.ColorGrayToBGR)
	gocv.CvtColor(dilateImg, &dilateImg, gocv.ColorGrayToBGR)
	gocv.BitwiseAnd(dilateImg, threshImg, &preResult)

	result := gocv.NewMat()
	defer result.Close()
	gocv.MorphologyExWithParams(preResult, &result, gocv.MorphOpen, kernel, 2, gocv.BorderConstant)

	// denoising
	final := gocv.NewMat()
	defer final.Close()
	gocv.BitwiseAnd(result, threshImg, &final)

	// optimize OCR by setting image to be black text and white bg
	gocv.CvtColor(final, &final, gocv.ColorBGRToGray)
	threshFinalImg := gocv.NewMat()
	defer threshFinalImg.Close()
	gocv.Threshold(final, &threshFinalImg, 0, 255, gocv.ThresholdOtsu+gocv.ThresholdBinary)

	whiteFinal := gocv.NewMatWithSize(threshFinalImg.Rows(), threshFinalImg.Cols(), threshFinalImg.Type())
	defer whiteFinal.Close()
	whiteFinal.SetTo(scalar)
	gocv.Subtract(whiteFinal, threshFinalImg, &threshFinalImg)

	gocv.IMWrite(imagePath, threshFinalImg)
}

func applyOCR(imagePath string) (text string, err error) {
	ocr := gosseract.NewClient()
	defer ocr.Close()
	err = ocr.SetImage(imagePath)
	throwError(err)
	text, err = ocr.Text()
	throwError(err)
	return text, nil
}

//////////////////
//  LEMNOS API  //
//////////////////

func getShorts(apiUrl string) (shorts []Short, pageToken string, err error) {
	fmt.Println("Querying:", apiUrl)
	request, err := http.NewRequest("GET", apiUrl, nil)
	throwError(err)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{}
	response, err := client.Do(request)
	throwError(err)
	responseBody, err := io.ReadAll(response.Body)
	throwError(err)

	var formattedData Message
	err = json.Unmarshal(responseBody, &formattedData)
	throwError(err)
	defer response.Body.Close()
	item := formattedData.Items[0]
	return item.Shorts, item.NextPageToken, nil
}

func isShortInDB(short Short) (found bool) {
	if strings.Contains(short.Title, "#millionaireinthemaking") || isDate(short.Title) || short.Title == "#millionareinthemaking" {
		var expectedShort models.Short
		result := database.DB.Db.First(&expectedShort, "video_id = ?", short.VideoID)
		return !errors.Is(result.Error, gorm.ErrRecordNotFound)
	}
	return true
}

func insertShort(short Short) (err error) {
	text, err := extractIncome(short.VideoID)
	throwError(err)
	if strings.Contains(short.Title, "#millionaireinthemaking") || isDate(short.Title) || short.Title == "#millionareinthemaking" {
		title := verifyNumberData(short.Title, "title")

		revenue := verifyNumberData(text, "revenue")
		expenses := verifyNumberData(text, "expenses")

		newShort := models.Short{Title: title, VideoID: short.VideoID, Revenue: revenue, Expenses: expenses, NetResult: revenue - expenses}
		database.DB.Db.FirstOrCreate(&newShort, "video_id = ?", short.VideoID) // IFF record doesn't exist already
	}
	return nil
}

/////////////
//  UTILS  //
/////////////

func verifyNumberData(text string, dataType string) (num int) {
	var check *regexp.Regexp
	switch dataType {
	case "revenue":
		check = revenueRegex
	case "expenses":
		check = expensesRegex
	case "title":
		check = titleRegex
	default:
		return -123456789
	}
	match := check.FindStringSubmatch(text)
	if len(match) > 1 {
		num, err := strconv.Atoi(match[1])
		throwError(err)
		return num
	}
	return -123456789
}

func isDate(str string) bool {
	layout := "January 2, 2006"
	_, err := time.Parse(layout, str)
	return err == nil
}

func throwError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
