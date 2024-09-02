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

type Short struct {
	VideoID string
	Title   string
}

type Item struct {
	Shorts        []Short
	NextPageToken string
}

type Message struct {
	Items []Item
}

// //////////////////
//	DB OPERATIONS  //
// //////////////////

func populateShorts(apiUrl string) {
	shorts, pageToken, err := getShorts(apiUrl)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, short := range shorts {
		err = insertShort(short)
		if err != nil {
			fmt.Println(err)
		}
	}

	for len(pageToken) > 0 {
		newPageApiUrl := apiUrl + "&pageToken=" + pageToken
		shorts, newPageToken, err := getShorts(newPageApiUrl)
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, short := range shorts {
			result := isShortInDB(short)
			if err != nil {
				fmt.Println(err)
			}
			if result {
				return
			} else {
				err = insertShort(short)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		pageToken = newPageToken
	}
}

///////////////////
//  PARSE VIDEO  //
///////////////////

func extractIncome(videoID string) (text string, err error) {
	URL, err := getVideoData(videoID)
	if err != nil {
		return "", err
	}
	imagePath, err := downloadLastFrame(videoID, URL)
	if err != nil {
		return "", err
	}
	preProcessImage(imagePath)
	text, err = applyOCR(imagePath)
	if err != nil {
		return "", err
	}
	return text, nil
}

func getVideoData(videoID string) (URL string, err error) {
	fmt.Println("Downloading video data from video id:", videoID)
	yt := youtube.Client{}
	video, err := yt.GetVideoContext(context.Background(), videoID)
	if err != nil {
		return "", err
	}
	format := video.Formats[0] // best quality video is always first (easiest for us to parse)
	return format.URL, nil
}

func downloadLastFrame(videoID string, URL string) (imagePath string, err error) {
	fmt.Println("Downloading the last frame image from video id:", videoID)
	imagePath = "./out/" + videoID + ".png"
	err = ffmpeg.Input(URL).
		Filter("reverse", ffmpeg.Args{}).
		Output(imagePath, ffmpeg.KwArgs{"frames:v": 1}).
		OverWriteOutput().
		ErrorToStdOut().
		Run()
	return imagePath, err
}

func preProcessImage(imagePath string) {
	fmt.Println("Performing pre-processing on the image:", imagePath)
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
	fmt.Println("Apply optical character recognition on the image:", imagePath)
	ocr := gosseract.NewClient()
	defer ocr.Close()
	err = ocr.SetImage(imagePath)
	if err != nil {
		return "", err
	}
	text, err = ocr.Text()
	if err != nil {
		return "", err
	}
	return text, nil
}

//////////////////
//  LEMNOS API  //
//////////////////

func getShorts(apiUrl string) (shorts []Short, pageToken string, err error) {
	fmt.Println("Querying the API:", apiUrl)
	request, _ := http.NewRequest("GET", apiUrl, nil)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return []Short{}, "", fmt.Errorf("Failed to send a request to %s. Receiving %s", apiUrl, err)
	}
	responseBody, _ := io.ReadAll(response.Body)

	var formattedData Message
	err = json.Unmarshal(responseBody, &formattedData)
	if err != nil {
		return []Short{}, "", fmt.Errorf("JSON response received from %s is ill-formed. Receiving %s", apiUrl, err)
	}
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
	if err != nil {
		return err
	}
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
		check = regexp.MustCompile(`\+\s\$(\d+)\s\(revenue\)`)
	case "expenses":
		check = regexp.MustCompile(`\-\s\$(\d+)\s\(expenses\)`)
	case "title":
		check = regexp.MustCompile(`Day\s(\d+)\s#millionaireinthemaking`)
	default:
		return -123456789
	}
	match := check.FindStringSubmatch(text)
	if len(match) > 1 {
		num, err := strconv.Atoi(match[1])
		if err != nil {
			fmt.Println(err)
		}
		return num
	}
	return -123456789
}

func isDate(str string) bool {
	layout := "January 2, 2006"
	_, err := time.Parse(layout, str)
	return err == nil
}
