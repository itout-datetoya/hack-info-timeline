package main

import (
	"context"
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

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// 設定の読み込
	dbConnStr := os.Getenv("DATABASE_URL")

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	telegramAppIDStr := os.Getenv("TELEGRAM_APP_ID")
	telegramAppHash := os.Getenv("TELEGRAM_APP_HASH")
	telegramPhoneNumber := os.Getenv("TELEGRAM_PHONE_NUMBER")
	telegramHash := os.Getenv("TELEGRAM_AUTH_HASH")
	teregramCode := os.Getenv("TELEGRAM_CODE")
	telegramHackingChannel := os.Getenv("TELEGRAM_HACKING_CHANNEL_USERNAME")
	telegramTransferChannel := os.Getenv("TELEGRAM_TRANSFER_CHANNEL_USERNAME")

	if telegramAppIDStr == "" || telegramAppHash == "" || telegramHackingChannel == "" ||
		 telegramTransferChannel == "" || telegramPhoneNumber == "" ||
		  geminiAPIKey == "" ||
		  	dbConnStr == ""{
		log.Fatal("user environment variables not fully set.")
	}
	telegramAppID, err := strconv.Atoi(telegramAppIDStr)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_APP_ID: %v", err)
	}

	m, err := migrate.New("file://migrations", dbConnStr)
	if err != nil {
		log.Fatalf("FATAL: failed to create migrate instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("FATAL: failed to run migrations: %v", err)
	}

	// データベース接続の初期化
	db, err := sqlx.Connect("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 依存性の注入 (DI)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	hackingRepo := datastore.NewHackingRepository(db)
	transferRepo := datastore.NewTransferRepository(db)

	// ToDo Telegram Client Managerの初期化と接続
	telegramClientManager := gateway.NewTelegramClientManager(
		telegramAppID,
		telegramAppHash,
	)

	// Runメソッドを呼び出して接続を開始
	if err := telegramClientManager.Run(ctx, telegramPhoneNumber, telegramHash, teregramCode); err != nil {
		log.Fatalf("Failed to run Telegram Gateway: %v", err)
	}
	log.Println("Telegram client connected and ready.")

	// 各gatewayの初期化
	telegramHackingGateway := gateway.NewTelegramHackingPostGateway(
		telegramClientManager,
		telegramHackingChannel,
	)
	telegramTransferGateway := gateway.NewTelegramTransferPostGateway(
		telegramClientManager,
		telegramTransferChannel,
	)
	geminiGateway, err := gateway.NewGeminiGateway(ctx, geminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize Gemini Gateway: %v", err)
	}

	// 各ハンドラーの初期化
	hackingUsecase := usecases.NewHackingUsecase(hackingRepo, telegramHackingGateway, geminiGateway)
	transferUsecase := usecases.NewTransferUsecase(transferRepo, telegramTransferGateway)
	hackingHandler := if_http.NewHackingHandler(hackingUsecase)
	transferHandler := if_http.NewTransferHandler(transferUsecase)

	// 5分毎のTickerを作成
	ticker := time.NewTicker(5 * time.Minute)

	// 定期実行処理
	go func() {
		// サーバー起動時に一度即時実行
		log.Println("Initial scraping process started...")
		initialScrapeCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)

		if _, errs := hackingUsecase.ScrapeAndStore(initialScrapeCtx); len(errs) > 0 {
			log.Printf("Initial hacking info scraping finished with errors: %v", errs)
		} else {
			log.Println("Initial hacking info finished successfully.")
		}

		if _, errs := transferUsecase.ScrapeAndStore(initialScrapeCtx); len(errs) > 0 {
			log.Printf("Initial transfer info scraping finished with errors: %v", errs)
		} else {
			log.Println("Initial transfer info scraping finished successfully.")
		}

		cancel()

		// Tickerとシャットダウンシグナルを待機
		for {
			select {
			case <-ticker.C:
				log.Println("Periodic scraping process started...")
				
				scrapeCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
				if _, errs := hackingUsecase.ScrapeAndStore(scrapeCtx); len(errs) > 0 {
					log.Printf("Periodic hacking info scraping finished with errors: %v", errs)
				} else {
					log.Println("Periodic hacking info scraping finished successfully.")
				}

				if _, errs := transferUsecase.ScrapeAndStore(scrapeCtx); len(errs) > 0 {
					log.Printf("Periodic transfer info scraping finished with errors: %v", errs)
				} else {
					log.Println("Periodic transfer info scraping finished successfully.")
				}

				cancel()

			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	// ルーターとHTTPサーバーのセットアップ
	router := if_http.NewRouter(*hackingHandler, *transferHandler)
	srv := &http.Server{
		Addr:    ":10000",
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
	if err := telegramClientManager.Stop(); err != nil {
		log.Println("Failed to stop telegram client:", err)
	}

	// Geminiクライアントを停止
	if err := geminiGateway.Stop(); err != nil {
		log.Println("Failed to stop gemini client:", err)
	}

	log.Println("Server exiting")
}