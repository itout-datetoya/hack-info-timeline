package gateway

import (
	"context"
)

// ハッキング情報の投稿
type HackingPost struct {
	Text		string
	Network		string
	Amount		string
	TxHash		string
}

// 抽出されたハッキング情報
type ExtractedHackingInfo struct {
	Protocol	string
	Network		string
	Amount		string
	TxHash		string
	TagNames	[]string
}

// Telegram APIとのハッキング情報の通信を抽象化
type TelegramHackingPostGateway interface {
	GetPosts(ctx context.Context) ([]*HackingPost, error)
}