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

// 送金情報の投稿
type TransferPost struct {
	Token		string
	Amount		string
	From		string
	To			string
	TagNames	[]string
}

// 抽出されたハッキング情報
type ExtractedHackingInfo struct {
	Protocol	string
	Network		string
	Amount		string
	TxHash		string
	Tokens		[]string
}

// Telegram APIとのハッキング情報の通信を抽象化
type TelegramHackingPostGateway interface {
	GetPosts(ctx context.Context) ([]*HackingPost, error)
}

// Telegram APIとの送金情報の通信を抽象化
type TelegramTransferPostGateway interface {
	GetPosts(ctx context.Context) ([]*TransferPost, error)
}

// Gemini APIとの通信を抽象化
type GeminiGateway interface {
	AnalyzeAndExtract(ctx context.Context, post *HackingPost) (*ExtractedHackingInfo, error)
}