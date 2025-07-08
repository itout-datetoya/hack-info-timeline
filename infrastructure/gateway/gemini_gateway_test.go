package gateway

import (
	"testing"

	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)


func TestAnalyzeAndExtract(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// 設定の読み込み
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	// 依存性の注入 (DI)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	geminiGateway, err := NewGeminiGateway(ctx, geminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize Gemini Gateway: %v", err)
	}

	post := &gateway.HackingPost{
		Text: "sUSDe (https://t.me/defimon_alerts/1379) and scrvUSD (https://t.me/defimon_alerts/1415) collateral branches of Asymmetry Finance (https://www.asymmetry.finance/)'s USDaf were shut down by an external party to gain 2% urgent redemption premiums.\n\nUnclear whether it's a whitehat operation or a hack, but earlier in June Asymmetry published a report on the USDaf oracle vulnerability (https://medium.com/@asymmetryfin/report-usdaf-oracle-incident-d40feff2ae52). The oracle bug boils down to an edge case when calculating price staleness from Chainlink which bypasses a fallback oracle. The report mentioned tBTC, sDAI and sUSDS collateral branches and urged users to unwind USDaf positions, however sUSDe and scrvUSD collateral branches remained affected by the oracle bug.\n\nThe attack requires landing a fetchPrice() tx at a block which is exactly 86400 seconds after a last Chainlink oracle price update to bypass the fallback oracle and shut down the trove. The patient attacker managed to perform this two times (noticeably not without errors (https://etherscan.io/tx/0x4616bcd9d4062322fa5aa79c7f9a795609578c2a836cf460f881d4ba7c909502)) and call urgentRedemption() on sUSDe and scrvUSD troves to gain 2% of the total pool value. \n\nAsymmetry Finance was notified of these transactions 🙏",
		Network: "mainnet",
		Amount: "$4,204.55",
		TxHash: "0xc3192361c65347c94935912188a94a923ff77da8",
	}

	// Geminiでテキストを分析
	extractedInfo, err := geminiGateway.AnalyzeAndExtract(ctx, post)
	if err != nil {
		t.Error(err)
	}

	stop()       // 他のコンテキストユーザーにキャンセルを通知
	log.Println("Shutting down server...")

	// Geminiクライアントを停止
	if err := geminiGateway.Stop(); err != nil {
		log.Println("Failed to stop gemini client:", err)
	}

	log.Println("Server exiting")

	t.Log(extractedInfo.Protocol, extractedInfo.Network,
			extractedInfo.Amount, extractedInfo.TxHash,
				extractedInfo.TagNames)

}