package gateway

import (
	"testing"

	"context"
	"log"
	"os"
	"path/filepath"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	_ "github.com/lib/pq"
)


func TestTransferGatewayGetPosts(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// 設定の読み込み
	telegramAppIDStr := os.Getenv("TELEGRAM_APP_ID")
	telegramAppHash := os.Getenv("TELEGRAM_APP_HASH")
	phone := os.Getenv("TELEGRAM_PHONE_NUMBER")
	hash := os.Getenv("TELEGRAM_AUTH_HASH")
	code := os.Getenv("TELEGRAM_CODE")
	telegramHackingChannel := os.Getenv("TELEGRAM_HACKING_CHANNEL_USERNAME")
	telegramTransferChannel := os.Getenv("TELEGRAM_TRANSFER_CHANNEL_USERNAME")

	if telegramAppIDStr == "" || telegramAppHash == "" || telegramHackingChannel == "" ||
		telegramTransferChannel == "" || phone == "" {
		log.Fatal("Telegram user client environment variables not fully set.")
	}
	telegramAppID, err := strconv.Atoi(telegramAppIDStr)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_APP_ID: %v", err)
	}

	// 依存性の注入 (DI)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()



	sessionDir := ".td"
	os.MkdirAll(sessionDir, 0755)
	client := telegram.NewClient(telegramAppID, telegramAppHash, telegram.Options{
		Logger: logger,
		SessionStorage: &session.FileStorage{
			Path: filepath.Join(sessionDir, "session.json"),
		},
	})
	telegramClientManager := &TelegramClientManager{client: client}

	// Runメソッドを呼び出して接続を開始
	if err := telegramClientManager.Run(ctx, phone, hash, code); err != nil {
		log.Fatalf("Failed to run Telegram Gateway: %v", err)
	}
	log.Println("Telegram client connected and ready.")

	// 各gatewayの初期化
	telegramTransferGateway := NewTelegramTransferPostGateway(
		telegramClientManager,
		telegramTransferChannel,
	)

	transferPosts, err := telegramTransferGateway.GetPosts(ctx)
	if err != nil {
		t.Error(err)
	}

	stop()       // 他のコンテキストユーザーにキャンセルを通知
	log.Println("Shutting down server...")

	// Telegramクライアントを停止
	if err := telegramClientManager.Stop(); err != nil {
		log.Println("Failed to stop telegram client:", err)
	}

	log.Println("Server exiting")
	t.Log(len(transferPosts))

	for i := 0; i < 5; i++ {
		t.Log(transferPosts[i])
	}

}