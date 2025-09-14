package gateway

import (
	"context"
	"time"
)

// ハッキング情報の投稿
type HackingPost struct {
	Text       string
	Network    string
	Amount     string
	TxHash     string
	ReportTime time.Time
	MessageID  int
}

// 抽出されたハッキング情報
type ExtractedHackingInfo struct {
	Protocol string
	Network  string
	Amount   string
	TxHash   string
	TagNames []string
}

// Telegram APIとのハッキング情報の通信を抽象化
type TelegramHackingPostGateway interface {
	SetLastMessageID(lastMessageID int)
	LastMessageID() int
	ChannelUsername() string
	GetPosts(ctx context.Context, limit int) ([]*HackingPost, error)
	GetPostsOver100(ctx context.Context, limit int) ([]*HackingPost, error)
}
