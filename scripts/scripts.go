package scripts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
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

type ShortMetadata struct {
	VideoID string
	Title   string
}

type Item struct {
	ShortMetadatas []ShortMetadata
	NextPageToken  string
}

type Message struct {
	Items []Item
}

// //////////////////
//	DB OPERATIONS  //
// //////////////////

func populateShortsEveryPage(apiUrl string) {
	pageToken, err := populateShorts(apiUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

	for len(pageToken) > 0 {
		newPageApiUrl := apiUrl + "&pageToken=" + pageToken
		newPageToken, err := populateShorts(newPageApiUrl)
		if err != nil {
			fmt.Println(err)
			return
		}
		pageToken = newPageToken
	}
}

func populateShorts(apiUrl string) (pageToken string, err error) {
	shortMetadatas, pageToken, err := getShortMetadatas(apiUrl)
	if err != nil {
		return "", err
	}
	for _, shortMetadata := range shortMetadatas {
		err = insertShort(shortMetadata)
		if err != nil {
			return "", err
		}
	}
	return pageToken, nil
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
	err = os.Remove(imagePath)
	if err != nil {
		return "", err
	}
	return text, nil
}

//////////////////
//  LEMNOS API  //
//////////////////

func getShortMetadatas(apiUrl string) (shortMetadataList []ShortMetadata, pageToken string, err error) {
	fmt.Println("Querying the API:", apiUrl)
	request, _ := http.NewRequest("GET", apiUrl, nil)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return []ShortMetadata{}, "", fmt.Errorf("Failed to send a request to %s. Receiving %s", apiUrl, err)
	}
	responseBody, _ := io.ReadAll(response.Body)

	var formattedData Message
	err = json.Unmarshal(responseBody, &formattedData)
	if err != nil {
		return []ShortMetadata{}, "", fmt.Errorf("JSON response received from %s is ill-formed. Receiving %s", apiUrl, err)
	}
	defer response.Body.Close()
	item := formattedData.Items[0]
	return item.ShortMetadatas, item.NextPageToken, nil
}

func insertShort(shortMetadata ShortMetadata) (err error) {
	fmt.Println("Attempting to insert the short, %s, into the database", shortMetadata.VideoID)
	text, err := extractIncome(shortMetadata.VideoID)
	if err != nil {
		return err
	}
	if strings.Contains(shortMetadata.Title, "#millionaireinthemaking") || isDate(shortMetadata.Title) || shortMetadata.Title == "#millionareinthemaking" {
		title := verifyNumberData(shortMetadata.Title, "title")
		revenue := verifyNumberData(text, "revenue")
		expenses := verifyNumberData(text, "expenses")

		newShort := models.Short{Title: title, VideoID: shortMetadata.VideoID, Revenue: revenue, Expenses: expenses, NetResult: revenue - expenses}

		result := database.DB.Db.First(&shortMetadata, "video_id = ?", shortMetadata.VideoID)
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			database.DB.Db.Create(&newShort) // IFF record doesn't exist already
			fmt.Println("Successfully inserted the short, %s, into the database", shortMetadata.VideoID)
			return nil
		} else {
			return fmt.Errorf("The short, %s, already exists in the database.", shortMetadata.VideoID)
		}
	}
	return fmt.Errorf("The short, %s, is not a #millionaireinthemaking video.", shortMetadata.VideoID)
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
