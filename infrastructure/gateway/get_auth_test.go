package gateway

import (
	"testing"

	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func TestGetAuth(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// 設定の読み込み
	telegramAppIDStr := os.Getenv("TELEGRAM_APP_ID")
	telegramAppHash := os.Getenv("TELEGRAM_APP_HASH")
	phone := os.Getenv("TELEGRAM_PHONE_NUMBER")
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

	log.Println("Client setting")

	// 認証情報を取得
	if err := telegramClientManager.getAuth(ctx, phone); err != nil {
		log.Fatalf("Failed to get auth code: %v", err)
	}

	log.Println("Get Auth")

	stop() // 他のコンテキストユーザーにキャンセルを通知
	log.Println("Shutting down server...")

	// Telegramクライアントを停止
	if err := telegramClientManager.Stop(); err != nil {
		log.Println("Failed to stop telegram client:", err)
	}

	log.Println("Server exiting")

}

func (m *TelegramClientManager) getAuthHash(ctx context.Context, phone string) error {

	sentCode, err := m.client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
	if err != nil {
		return err
	}
	log.Println("Send code")
	authSendCode, ok := sentCode.(*tg.AuthSentCode)
	if !ok {
		return fmt.Errorf("failed to get auth code")
	}
	hash := authSendCode.PhoneCodeHash
	log.Printf("Hash: %s", hash)

	return nil
}

func (m *TelegramClientManager) getAuth(ctx context.Context, phone string) error {
	ctx, m.stop = context.WithCancel(ctx)

	ready := make(chan struct{})
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()
		err := m.client.Run(ctx, func(ctx context.Context) error {
			// 認証状態を確認
			status, err := m.client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to get auth status: %w", err)
			}

			// 未認証の場合、電話番号で認証フローを開始
			if !status.Authorized {
				if err := m.getAuthHash(ctx, phone); err != nil {
					return fmt.Errorf("failed auth flow: %w \n please set sent auth code in telegram message", err)
				}
			}

			// APIクライアントを取得して保持
			m.api = m.client.API()

			// 準備完了を通知
			close(ready)

			// サーバーがシャットダウンするまで待機
			<-ctx.Done()
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			// 準備完了前にエラーが発生した場合
			fmt.Fprintf(os.Stderr, "gotd client run error: %v\n", err)
			// readyがクローズされていない場合、クローズしてエラーを通知
			select {
			case <-ready:
			default:
				close(ready)
			}
		}
	}()

	// 準備が完了するか、コンテキストがキャンセルされるまで待機
	select {
	case <-ready:
		if m.api == nil {
			return errors.New("failed to initialize telegram client")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
