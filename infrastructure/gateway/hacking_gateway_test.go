package gateway

import (
	"testing"

	"context"
	"fmt"
	dm_gateway "github.com/itout-datetoya/hack-info-timeline/domain/gateway"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func TestHackingGatewayGetPosts(t *testing.T) {
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
	telegramHackingChannels := strings.Split(os.Getenv("TELEGRAM_HACKING_CHANNEL_USERNAMES"), ",")
	telegramTransferChannels := strings.Split(os.Getenv("TELEGRAM_TRANSFER_CHANNEL_USERNAMES"), ",")

	if telegramAppIDStr == "" || telegramAppHash == "" || telegramHackingChannels[0] == "" ||
		telegramTransferChannels[0] == "" || phone == "" {
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
	var telegramHackingGateways []dm_gateway.TelegramHackingPostGateway
	for _, channel := range telegramHackingChannels {
		telegramHackingGateways = append(telegramHackingGateways,
			NewTelegramHackingPostGateway(
				telegramClientManager,
				channel,
			))
	}

	limit := 100

	var wg sync.WaitGroup
	errsChan := make(chan error, len(telegramHackingGateways))
	var posts []*dm_gateway.HackingPost
	var mu sync.Mutex

	for _, gw := range telegramHackingGateways {
		wg.Add(1)
		go func(gw dm_gateway.TelegramHackingPostGateway) {
			defer wg.Done()
			newPosts, err := gw.GetPosts(ctx, limit)
			if err != nil {
				errsChan <- fmt.Errorf("failed to get posts from telegram: %w", err)
				return
			}
			mu.Lock()
			posts = append(posts, newPosts...)
			mu.Unlock()
		}(gw)
	}

	wg.Wait()
	close(errsChan)

	var getPostsErrors []error
	for err := range errsChan {
		getPostsErrors = append(getPostsErrors, err)
	}

	if len(getPostsErrors) != 0 {
		log.Fatalf("failed to get posts from telegram: %v", getPostsErrors)
	}

	stop() // 他のコンテキストユーザーにキャンセルを通知
	log.Println("Shutting down server...")

	// Telegramクライアントを停止
	if err := telegramClientManager.Stop(); err != nil {
		log.Println("Failed to stop telegram client:", err)
	}

	log.Println("Server exiting")

	for _, post := range posts {
		t.Log(post)
	}
}
