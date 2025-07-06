package gateway

import (
	"context"
	"fmt"
	"strings"
	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type geminiGateway struct {
	client *genai.Client
	model *genai.GenerativeModel
}

// genai ライブラリを使ってクライアントを初期化
func NewGeminiGateway(ctx context.Context, apiKey string) (gateway.GeminiGateway, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is missing")
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	// 使用するモデルを指定
	model := client.GenerativeModel("gemini-2.5-flash")
	return &geminiGateway{client: client, model: model}, nil
}

func (g *geminiGateway) Stop() error {
	return g.client.Close()
}

func (g *geminiGateway) AnalyzeAndExtract(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error) {
	var extractedInfo gateway.ExtractedHackingInfo
	
	// プロトコル名抽出用のプロンプト
	protocolNamePrompt := genai.Text(fmt.Sprintf(`
		You are a specialized AI assistant for DeFi security analysis. Your task is to extract the name of the hacked DeFi protocol from the provided text and provide both the original and a cleaned version of the name.
	
		Follow these rules strictly:
		1. Identify the primary DeFi protocol that was hacked.
		2. For the first part of the output, extract the name exactly as it appears in the text.
		3. For the second part of the output, create a cleaned version by:
    		a. Converting the name to lowercase.
    		b. Removing generic suffixes and domain extensions. This includes parts like .fi, .finance, .protocol, and any top-level domain (e.g., .trade, .exchange, .xyz). The goal is to get the core name.
		4. Return a single line of text with the original name and the cleaned name separated by a comma. Do not add any spaces around the comma.
		5. The required format is: OriginalName,cleanedname

		For example:
		- Text: "Attack on Resupply.fi" -> Response: Resupply.fi,resupply
		- Text: "Sonne Finance was exploited" -> Response: Sonne Finance,sonne
		- Text: "The Onyx Protocol hack" -> Response: Onyx Protocol,onyx
		- Text: "An exploit on gradient.trade..." -> Response: gradient.trade,gradient

		Now, analyze the following text and provide the response in the specified format.

		Text:
		"%s"
	`, post.Text))

	// geminiAPI呼び出し(プロトコル名)
	protocolNameResp, err := g.model.GenerateContent(ctx, protocolNamePrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content from Gemini API: %w", err)
	}

	if len(protocolNameResp.Candidates) == 0 || len(protocolNameResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("invalid response structure from Gemini API")
	}

	protocolNamePart := protocolNameResp.Candidates[0].Content.Parts[0]
	protocolNamesStr, ok := protocolNamePart.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response part type: %T", protocolNamePart)
	}

	// カンマで分割
	protocolNames := strings.Split(string(protocolNamesStr), ",")

	// トークン名抽出用のプロンプト
	tokenPrompt := genai.Text(fmt.Sprintf(`
		You are a specialized AI assistant for DeFi and crypto token analysis. Your task is to identify and list the ticker symbols of all tokens directly mentioned in the context of a hack or exploit from the provided text.

		Follow these instructions carefully:
		1.  Identify the ticker symbols of the cryptocurrencies involved. Ticker symbols are short, often all-caps or prefixed abbreviations (e.g., ETH, WBTC, CRV, wstETH).
		2.  List only the tokens that were directly stolen, manipulated, or used as part of the exploit.
		3.  Do not include protocol names (e.g., 'Sonne', 'Onyx'), general currency symbols (e.g., '$', '€'), or irrelevant acronyms.
		4.  Return the tickers as a single, comma-separated string without spaces. For example: "TICKER1,TICKER2,TICKER3".
		5.  If no specific token tickers are mentioned as being involved in the hack, return the exact text "N/A".

		---
		**Example 1:**
		Text: "An old but still relevant ERC4626 first deposit attack caused a multimillion loss for the protocol. A new wstUSR market was deployed which used an empty crvUSD Curve Vault... an address exploited the new market to drain 9.3 million $."
		Response: wstUSR,crvUSD

		**Example 2:**
		Text: "The attacker manipulated the price oracle for the FTM token on the Geist Finance protocol, allowing them to borrow other assets cheaply."
		Response: FTM

		**Example 3:**
		Text: "A vulnerability was found in the smart contract of a lending protocol. Thankfully, the whitehat hacker notified the team and no funds were lost."
		Response: N/A
		---

		Now, analyze the following text and provide the response.

		Text:
		"%s"
	`, post.Text))

	// geminiAPI呼び出し(トークン名)
	tokenResp, err := g.model.GenerateContent(ctx, tokenPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content from Gemini API: %w", err)
	}

	if len(tokenResp.Candidates) == 0 || len(tokenResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("invalid response structure from Gemini API")
	}

	tokenPart := tokenResp.Candidates[0].Content.Parts[0]
	tokensStr, ok := tokenPart.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response part type: %T", protocolNamePart)
	}

	// 表記ゆれ防止のため小文字にしてカンマで分割
	tokens := strings.Split(strings.ToLower(string(tokensStr)), ",")

	extractedInfo.Protocol = protocolNames[0]
	extractedInfo.Network = post.Network
	extractedInfo.Amount = post.Amount
	extractedInfo.TxHash = post.TxHash

	// 表記ゆれ防止のため小文字化
	if tokens[0] != "N/A" {
		extractedInfo.TagNames = append(tokens, strings.ToLower(protocolNames[1]))
	} else {
		extractedInfo.TagNames = []string{strings.ToLower(protocolNames[1])}
	}



	return &extractedInfo, nil
}