package gateway

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"strings"
	"log"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// gotdクライアント接続を管理する構造体
type TelegramClientManager struct {
	client *telegram.Client
	api    *tg.Client
	wg     sync.WaitGroup
	stop   context.CancelFunc
}

// gotdクライアントをセットアップして、TelegramClientManagerを生成
func NewTelegramClientManager(appID int, appHash string) *TelegramClientManager {
	sessionDir := ".td"
    os.MkdirAll(sessionDir, 0755)

	client := telegram.NewClient(appID, appHash, telegram.Options{
		SessionStorage: &session.FileStorage{
			Path: filepath.Join(sessionDir, "session.json"),
		},
	})
	return &TelegramClientManager{client: client}
}

// クライアントをバックグラウンドで実行し、接続と認証を処理
// 接続が確立されるまでブロック
func (m *TelegramClientManager) Run(ctx context.Context, phone string, hash string, password string) error {
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
				if err := m.authFlow(ctx, phone, hash, password); err != nil {
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

// 認証情報が設定されていれば、認証を実行
// 認証情報が設定されていなければ、認証情報を取得してサーバーを停止
func (m *TelegramClientManager) authFlow(ctx context.Context, phone string, hash string, code string) error {
	if hash == "" || code == ""{
		sentCode, err := m.client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
		if err != nil {
			return err
		}
		log.Println("Send code")
		authSendCode, ok :=sentCode.(*tg.AuthSentCode)
		if !ok {
			return fmt.Errorf("failed to get auth code")
		}
		hash := authSendCode.PhoneCodeHash
		log.Printf("Hash: %s", hash)
		return fmt.Errorf("check code in telegram\nAuth Hash: %s", hash)

	} else {
		_, err := m.client.Auth().SignIn(ctx, phone, strings.TrimSpace(code), hash)
		if err != nil {
			return err
		}
	}

	return nil
}

// クライアントの接続を安全に停止
func (m *TelegramClientManager) Stop() error {
	if m.stop != nil {
		m.stop()
		m.wg.Wait()
	}
	return nil
}

// 認証済みの *tg.Client を返す
func (m *TelegramClientManager) API() *tg.Client {
	return m.api
}
