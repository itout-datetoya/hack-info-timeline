package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"errors"
	"github.com/itout-datetoya/hack-info-timeline/infrastructure/datastore"
	"github.com/itout-datetoya/hack-info-timeline/infrastructure/gateway"
	if_http "github.com/itout-datetoya/hack-info-timeline/interfaces/http"
	"github.com/itout-datetoya/hack-info-timeline/usecases"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// 設定の読み込み
	// ToDO:DB, Gemini関連
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	telegramAppIDStr := os.Getenv("TELEGRAM_APP_ID")
	telegramAppHash := os.Getenv("TELEGRAM_APP_HASH")
	telegramChannel := os.Getenv("TELEGRAM_CHANNEL_USERNAME")

	if telegramAppIDStr == "" || telegramAppHash == "" || telegramChannel == "" {
		log.Fatal("Telegram user client environment variables not fully set.")
	}
	telegramAppID, err := strconv.Atoi(telegramAppIDStr)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_APP_ID: %v", err)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	// データベース接続の初期化
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 依存性の注入 (DI)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	hackingRepo := datastore.NewHackingRepository(db)

	// ToDo Telegram Client Managerの初期化と接続
	telegramGateway, err := gateway.NewTelegramGateway(
		telegramAppID,
		telegramAppHash,
		telegramChannel,
	)
	if err != nil {
		log.Fatalf("Failed to create Telegram Gateway: %v", err)
	}
	// Runメソッドを呼び出して接続を開始
	if err := telegramGateway.(*gateway.telegramGateway).Run(ctx); err != nil {
		log.Fatalf("Failed to run Telegram Gateway: %v", err)
	}
	log.Println("Telegram client connected and ready.")

	// ToDo Telegram Hacking Gatewayの初期化

	// ToDo Telegram Transfer Gatewayの初期化

	// ToDo Gemini Gatewayの初期化
	geminiGateway, err := gateway.NewGeminiGateway(ctx, geminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize Gemini Gateway: %v", err)
	}

	hackingUsecase := usecase.NewHackingUsecase(hackingRepo, telegramGateway, geminiGateway)
	hackingHandler := if_http.NewHackingHandler(hackingUsecase)

	// ルーターとHTTPサーバーのセットアップ
	router := if_http.NewRouter(hackingHandler)
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// HTTPサーバーを起動
	go func() {
		log.Println("Server starting on port 8080...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// シャットダウン処理
	<-ctx.Done() // SIGINT/SIGTERM を待機
	stop()       // 他のコンテキストユーザーにキャンセルを通知
	log.Println("Shutting down server...")

	// HTTPサーバーをグレースフルシャットダウン
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	// Telegramクライアントを停止
	if err := telegramGateway.Stop(); err != nil {
		log.Println("Failed to stop telegram client:", err)
	}

	log.Println("Server exiting")
}