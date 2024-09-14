package controller

import (
	"context"
	"fmt"
	"gin-fleamarket/dto"
	"gin-fleamarket/models"
	"gin-fleamarket/services"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

type IHanabiController interface {
	FindAll(ctx *gin.Context)
	FindByID(ctx *gin.Context)
	Create(ctx *gin.Context)
}

type HanabiController struct {
	services services.IHanabiService
}

func NewHanabiController(service services.IHanabiService) IHanabiController {
	return &HanabiController{services: service}
}

func (c *HanabiController) FindAll(ctx *gin.Context) {
	date := ctx.Query("date")

	hanabis, err := c.services.FindAll(date)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected error "})
		return
	}

	//ctx.JSON(http.StatusOK, gin.H{"All Hanabis": hanabis})
	ctx.JSON(http.StatusOK, hanabis)

}

func (c *HanabiController) FindByID(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	userId := user.(*models.User).ID
	hanabiId, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id "})
		return
	}

	foundedHanabi, err := c.services.FindByID(uint(hanabiId), uint(userId))
	if err != nil {
		if err.Error() == "hanabis not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected error"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"founded hanabi": foundedHanabi})
}

func (c *HanabiController) Create(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// ファイルを取得
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "File upload error"})
		return
	}
	defer file.Close()

	// 一意なファイル名を生成
	objectName := uuid.New().String() + "_" + header.Filename

	// Google Cloud Storage にアップロード
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("GCS_BUCKET_NAME environment variable is not set")
	}
	uploadedFileURL, err := uploadFileToGCS(bucketName, objectName, file)
	if err != nil {
		log.Printf("Failed to upload file: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// フォームデータを受け取る
	var input dto.CreateHanabiInput
	input.Name = ctx.PostForm("name")
	input.Description = ctx.PostForm("description")
	input.Tag = ctx.PostForm("tag")
	input.PhotoURL = uploadedFileURL

	// データのバリデーションを手動で行う
	if input.Name == "" || input.Description == "" || input.Tag == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	// Hanabiの作成処理
	userId := user.(*models.User).ID
	newHanabi, err := c.services.Create(input, userId)
	if err != nil {
		log.Printf("Failed to create Hanabi: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// ユーザー情報をプリロード
	if err := c.services.PreloadUser(newHanabi); err != nil {
		log.Printf("Failed to preload user data: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"AllHanabi": newHanabi})
}

// Google Cloud Storage にファイルをアップロードする関数
// func uploadFileToGCS(bucketName, objectName string, file multipart.File) (string, error) {
// 	ctx := context.Background()

// 	// 認証情報のJSONファイルのパス
// 	credentialsFile := "/Users/冨澤 健心/Documents/secret-key/team17_sokuseki/atomic-life-435113-t3-49a446fb681d.json" // ここにダウンロードしたJSONファイルのパスを指定

// 	// 認証情報を使用してクライアントを作成
// 	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
// 	if err != nil {
// 		return "", err
// 	}
// 	defer client.Close()

// 	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
// 	defer cancel()

// 	// バケットを指定してファイルをアップロード
// 	bucket := client.Bucket(bucketName)
// 	wc := bucket.Object(objectName).NewWriter(ctx)
// 	if _, err = io.Copy(wc, file); err != nil {
// 		return "", err
// 	}
// 	if err := wc.Close(); err != nil {
// 		return "", err
// 	}

// 	// 公開URLを作成
// 	uploadedFileURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
// 	return uploadedFileURL, nil
// }

// Google Cloud Storage にファイルをアップロードする関数
func uploadFileToGCS(bucketName, objectName string, file multipart.File) (string, error) {
	ctx := context.Background()

	// 環境変数から認証情報ファイルのパスを取得
	credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsFile == "" {
		return "", fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	// 認証情報を使用してクライアントを作成
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return "", err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// バケットを指定してファイルをアップロード
	bucket := client.Bucket(bucketName)
	wc := bucket.Object(objectName).NewWriter(ctx)
	if _, err = io.Copy(wc, file); err != nil {
		return "", err
	}
	if err := wc.Close(); err != nil {
		return "", err
	}

	// 公開URLを作成
	uploadedFileURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
	return uploadedFileURL, nil
}
