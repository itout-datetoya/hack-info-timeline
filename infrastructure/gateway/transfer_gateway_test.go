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
	"errors"
	"strings"
	"regexp"
	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"

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

func TestParseTransferMessage(t *testing.T) {
	testString := `⚠️⚠️⚠️141,271.0 #USDT transferred from Guarantee-Merchant to TUtjxCskyxs4WbPP1bT7GCA4zsZUVaHqHn.

Go MistTrack (https://light.misttrack.io/address/USDT-TRC20/TTg3UM69pbWTq6HMEAWjwAZUYci3L1TgfX) | Transaction Details (https://tronscan.org/#/transaction/5b3f4fbc336d12d2554657dd10afb56ac80cd5078067b542bb7582a4ea0b51d1)`

	post, err := parseTransferMessage(testString)
	if err != nil {
		t.Error(err)
	}

	t.Log(post)

}

func parseTransferMessage(message string) (*gateway.TransferPost, error) {
	// スペースで分割
	tokens := strings.Fields(message)

	found := false
	var post gateway.TransferPost
	var amount string

	// "transferred" を基準にパース
	for i, token := range tokens {
		if token == "transferred" && i > 1 && i+3 < len(tokens) {
			// "transferred" の前の単語が「送金額」と「トークン」
			amount = strings.ReplaceAll(tokens[i-2], ",", "")
			post.Token = strings.TrimPrefix(tokens[i-1], "#")

			// "transferred" の後の単語が "from", "送金元", "to", "送金先"
			if tokens[i+1] == "from" && tokens[i+3] == "to" {
				post.From = strings.TrimPrefix(tokens[i+2], "#")
				post.To = strings.TrimPrefix(tokens[i+4], "#")
				found = true
				break
			}
		}
	}

	re := regexp.MustCompile(`^[^0-9]+`)
	post.Amount = re.ReplaceAllString(amount, "")

	if !found {
		return nil, errors.New("TransferPost pattern not found in message")
	}

	return &post, nil
}