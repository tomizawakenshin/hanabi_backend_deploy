package main

import (
	"gin-fleamarket/controller"
	"gin-fleamarket/infra"
	"gin-fleamarket/middlewares"
	"gin-fleamarket/models"
	"net/http"
	"os" // 追加: 環境変数からポートを取得するため
	"time"

	"gin-fleamarket/reposotories"
	"gin-fleamarket/services"

	"log" // 追加: ログ出力のため

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupRouter(db *gorm.DB) *gin.Engine {

	authRepository := reposotories.NewAuthRepository(db)
	authService := services.NewAuthService(authRepository)
	authController := controller.NewAuthController(authService)

	hanabiRepository := reposotories.NewHanabiRepository(db)
	hanabiService := services.NewHanabiService(hanabiRepository)
	hanabiController := controller.NewHanabiController(hanabiService)

	commentRepository := reposotories.NewCommentMemoryRepository(db)
	commentService := services.NewCommentService(commentRepository, hanabiRepository)
	commentController := controller.NewCommentController(commentService)

	likeRepository := reposotories.NewLikeRepository(db)
	likeService := services.NewLikeService(likeRepository)
	likeController := controller.NewLikeController(likeService)

	r := gin.Default()
	// CORS 設定
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://your-frontend.vercel.app", "https://team17-frontend.vercel.app"}, // 新しいフロントエンドのドメインを追加
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},                                                         // 許可するHTTPメソッド
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},                                                         // 許可するリクエストヘッダー
		ExposeHeaders:    []string{"Content-Length"},                                                                                  // クライアントに公開するレスポンスヘッダー
		AllowCredentials: true,                                                                                                        // 認証情報（クッキーなど）の送信を許可
		MaxAge:           48 * time.Hour,                                                                                              // プリフライトリクエストのキャッシュ時間
	}))

	r.OPTIONS("/hanabi/getAll", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Status(http.StatusNoContent) // 204 No Contentを返す
	})

	//hanabiのエンドポイント
	hanabiRouterWithAuth := r.Group("/hanabi", middlewares.AuthMiddleware(authService))
	hanabiRouterWithAuth.POST("/create", hanabiController.Create)
	hanabiRouterWithAuth.GET("/getAll", hanabiController.FindAll)
	hanabiRouterWithAuth.GET("/getByID/:id", hanabiController.FindByID)

	//commentのエンドポイント
	commentRouterWithAuth := r.Group("/comment", middlewares.AuthMiddleware(authService))
	commentRouterWithAuth.POST("/create/:hanabiId", commentController.Create)

	//likeのエンドポイント
	likeRouterWithAuth := r.Group("/like", middlewares.AuthMiddleware(authService))
	likeRouterWithAuth.POST("/like/:commentId", likeController.Like)
	likeRouterWithAuth.DELETE("/unlike/:commentId", likeController.Unlike)

	//user認証関連のエンドポイント
	authRouter := r.Group("/auth")
	authRouter.POST("/signup", authController.SignUp)
	authRouter.POST("/login", authController.Login)

	return r
}

func main() {
	infra.Initialize()
	db := infra.SetupDB()

	if err := db.AutoMigrate(&models.User{}, &models.Comment{}, &models.Hanabi{}, &models.Like{}); err != nil {
		panic("Failed to migrate db")
	}

	r := setupRouter(db)

	// 環境変数からポートを取得し、Railwayのようなクラウド環境で動作するようにする
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // 環境変数が設定されていない場合はデフォルトで8080を使用
	}

	log.Printf("Server is running on port %s", port)
	r.Run("0.0.0.0:" + port) // 0.0.0.0を使用して外部からアクセス可能にする
}
