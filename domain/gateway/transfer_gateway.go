package gateway

import (
	"context"
	"time"
)

// 送金情報の投稿
type TransferPost struct {
	Token      string
	Amount     string
	From       string
	To         string
	ReportTime time.Time
	TagNames   []string
}

// Telegram APIとの送金情報の通信を抽象化
type TelegramTransferPostGateway interface {
	GetPosts(ctx context.Context, limit int) ([]*TransferPost, error)
}
