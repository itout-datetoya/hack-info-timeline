package main

import (
	"context"
	"errors"
	dm_gateway "github.com/itout-datetoya/hack-info-timeline/domain/gateway"
	"github.com/itout-datetoya/hack-info-timeline/infrastructure/datastore"
	"github.com/itout-datetoya/hack-info-timeline/infrastructure/gateway"
	if_http "github.com/itout-datetoya/hack-info-timeline/interfaces/http"
	"github.com/itout-datetoya/hack-info-timeline/usecases"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/patrickmn/go-cache"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// 設定の読み込
	dbConnStr := os.Getenv("DATABASE_URL")

	jsonString := os.Getenv("SESSION_JSON")

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	telegramAppIDStr := os.Getenv("TELEGRAM_APP_ID")
	telegramAppHash := os.Getenv("TELEGRAM_APP_HASH")
	telegramPhoneNumber := os.Getenv("TELEGRAM_PHONE_NUMBER")
	telegramHash := os.Getenv("TELEGRAM_AUTH_HASH")
	teregramCode := os.Getenv("TELEGRAM_CODE")
	telegramHackingChannels := strings.Split(os.Getenv("TELEGRAM_HACKING_CHANNEL_USERNAMES"), ",")
	telegramTransferChannels := strings.Split(os.Getenv("TELEGRAM_TRANSFER_CHANNEL_USERNAMES"), ",")

	if telegramAppIDStr == "" || telegramAppHash == "" || telegramHackingChannels[0] == "" ||
		telegramTransferChannels[0] == "" || telegramPhoneNumber == "" ||
		geminiAPIKey == "" ||
		dbConnStr == "" ||
		jsonString == "" {
		log.Fatal("User environment variables not fully set.")
		return
	}
	telegramAppID, err := strconv.Atoi(telegramAppIDStr)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_APP_ID: %v", err)
		return
	}

	m, err := migrate.New("file://migrations", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
		return
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
		return
	}

	// データベース接続の初期化
	db, err := sqlx.Connect("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		db.Close()
		return
	}
	defer db.Close()

	// キャッシュの初期化
	cache := cache.New(15*time.Minute, 20*time.Minute)

	dirPath := ".td"
	filePath := filepath.Join(dirPath, "session.json")

	os.MkdirAll(dirPath, 0755)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
	if err != nil {
		if os.IsExist(err) {
		} else {
			log.Fatalf("Failed to open file %v", err)
			return
		}
	} else {
		_, err = file.WriteString(jsonString)
		if err != nil {
			log.Fatalf("Failed to write file %v", err)
			return
		}
	}
	file.Close()

	// 依存性の注入 (DI)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	dbHackingRepo := datastore.NewDbHackingRepository(db)
	dbTransferRepo := datastore.NewDbTransferRepository(db)
	hackingRepo := datastore.NewHackingRepository(dbHackingRepo, cache)
	transferRepo := datastore.NewTransferRepository(dbTransferRepo, cache)

	// Telegram Client Managerの初期化と接続
	telegramClientManager := gateway.NewTelegramClientManager(
		telegramAppID,
		telegramAppHash,
	)

	// Runメソッドを呼び出して接続を開始
	if err := telegramClientManager.Run(ctx, telegramPhoneNumber, telegramHash, teregramCode); err != nil {
		log.Fatalf("Failed to run Telegram Gateway: %v", err)
		return
	}
	log.Println("Telegram client connected and ready.")

	// 各gatewayの初期化
	var telegramHackingGateways []dm_gateway.TelegramHackingPostGateway
	for _, channel := range telegramHackingChannels {
		telegramHackingGateways = append(telegramHackingGateways,
			gateway.NewTelegramHackingPostGateway(
				telegramClientManager,
				channel,
			))
	}

	var telegramTransferGateways []dm_gateway.TelegramTransferPostGateway
	for _, channel := range telegramTransferChannels {
		telegramTransferGateways = append(telegramTransferGateways,
			gateway.NewTelegramTransferPostGateway(
				telegramClientManager,
				channel,
			))
	}

	geminiGateway, err := gateway.NewGeminiGateway(ctx, geminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize Gemini Gateway: %v", err)
		return
	}

	// 各ハンドラーの初期化
	hackingUsecase := usecases.NewHackingUsecase(hackingRepo, telegramHackingGateways, geminiGateway)
	transferUsecase := usecases.NewTransferUsecase(transferRepo, telegramTransferGateways)
	hackingHandler := if_http.NewHackingHandler(hackingUsecase)
	transferHandler := if_http.NewTransferHandler(transferUsecase)

	// 10分毎のTickerを作成
	ticker := time.NewTicker(10 * time.Minute)

	// 定期実行処理
	go func() {
		// サーバー起動時に一度即時実行
		log.Println("Initial scraping process started...")
		initialScrapeCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)

		err = hackingUsecase.SetLastMessageIDToGateway(initialScrapeCtx)
		if err != nil {
			log.Printf("%v", err)
		}
		err = transferUsecase.SetLastMessageIDToGateway(initialScrapeCtx)
		if err != nil {
			log.Printf("%v", err)
		}

		if _, errs := hackingUsecase.ScrapeAndStore(initialScrapeCtx, 200); len(errs) > 0 {
			log.Printf("Initial hacking info scraping finished with errors: %v", errs)
		} else {
			log.Println("Initial hacking info finished successfully.")
		}

		if _, errs := transferUsecase.ScrapeAndStore(initialScrapeCtx, 200); len(errs) > 0 {
			log.Printf("Initial transfer info scraping finished with errors: %v", errs)
		} else {
			log.Println("Initial transfer info scraping finished successfully.")
		}

		err = hackingUsecase.SetTagToCache(initialScrapeCtx)
		if err != nil {
			log.Printf("%v", err)
		}

		cancel()

		// Tickerとシャットダウンシグナルを待機
		for {
			select {
			case <-ticker.C:
				log.Println("Periodic scraping process started...")

				scrapeCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
				if _, errs := hackingUsecase.ScrapeAndStore(scrapeCtx, 100); len(errs) > 0 {
					log.Printf("Periodic hacking info scraping finished with errors: %v", errs)
				} else {
					log.Println("Periodic hacking info scraping finished successfully.")
				}

				if _, errs := transferUsecase.ScrapeAndStore(scrapeCtx, 100); len(errs) > 0 {
					log.Printf("Periodic transfer info scraping finished with errors: %v", errs)
				} else {
					log.Println("Periodic transfer info scraping finished successfully.")
				}

				err := hackingUsecase.StoreLastMessageID(scrapeCtx)
				if err != nil {
					log.Printf("%v", err)
				}
				err = transferUsecase.StoreLastMessageID(scrapeCtx)
				if err != nil {
					log.Printf("%v", err)
				}

				err = hackingUsecase.SetTagToCache(scrapeCtx)
				if err != nil {
					log.Printf("%v", err)
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
