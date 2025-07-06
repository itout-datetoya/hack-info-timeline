package gateway

import (
	"context"
)

// Gemini APIとの通信を抽象化
type GeminiGateway interface {
	AnalyzeAndExtract(ctx context.Context, post *HackingPost) (*ExtractedHackingInfo, error)
	Stop() error
}