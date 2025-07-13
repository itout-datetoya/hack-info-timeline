package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"regexp"
	"time"
	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"

	"github.com/gotd/td/tg"
)

// TransferHackingPostGatewayを実装する構造体
type telegramTransferPostGateway struct {
	manager			*TelegramClientManager
	channelUsername	string
	channelPeer     *tg.InputPeerChannel
	lastMessageID	int
	mu            	sync.Mutex
}

// 新しいtelegramTransferPostGatewayを生成
func NewTelegramTransferPostGateway(manager *TelegramClientManager, channelUsername string) gateway.TelegramTransferPostGateway {
	return &telegramTransferPostGateway{manager: manager, channelUsername: channelUsername}
}

// 最後に取得した投稿以降、最新100件の投稿を取得
func (g *telegramTransferPostGateway) GetPosts(ctx context.Context) ([]*gateway.TransferPost, error) {
	api := g.manager.API()
	if api == nil {
		return nil, errors.New("telegram client is not ready")
	}

	// チャンネル名を解決してPeer情報を取得
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: g.channelUsername,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve username %s: %w", g.channelUsername, err)
	}
	channel, ok := resolved.Chats[0].(*tg.Channel)
	if !ok {
		return nil, fmt.Errorf("resolved peer is not a channel")
	}
	inputPeer := channel.AsInputPeer()
	g.channelPeer = inputPeer

	g.mu.Lock()
	defer g.mu.Unlock()
	// 最後に取得した投稿以降、最新100件の投稿を取得
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  inputPeer,
		MinID: g.lastMessageID, 
		Limit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get channel history: %w", err)
	}

	// 取得した投稿の内、送金情報を含むものをTransferPostに変換
	return g.convertMessages(history)
}

// 取得した投稿の内、送金情報を含むものをHackingPostに変換
func (g *telegramTransferPostGateway) convertMessages(history tg.MessagesMessagesClass) ([]*gateway.TransferPost, error) {
	// 取得したデータを投稿のスライスに変換
	channelMessages, ok := history.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, fmt.Errorf("failed to cast history to ChannelMessages")
	}

	var posts []*gateway.TransferPost
	for _, msg := range channelMessages.Messages {
		// チャンネルの投稿か確認
		if message, ok := msg.(*tg.Message); ok && message.Message != "" {
			
			// 投稿から送金情報を取得
			post, err := g.parseTransferMessage(message.Message)
			if err != nil {
				return nil, err
			}
			date := message.GetDate()
			post.ReportTime = time.Unix(int64(date), 0)

			// 投稿からタグを取得
			tagNames := g.extractTags(message.Message, message.Entities)
			post.TagNames = tagNames

			posts = append(posts, post)

			// 最後に取得した投稿のIDを更新
			if message.ID > g.lastMessageID {
				g.lastMessageID = message.ID
			}
		}
	}
	return posts, nil
}

// 投稿の形式からパースして送金情報を取得
func (g *telegramTransferPostGateway) parseTransferMessage(message string) (*gateway.TransferPost, error) {
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
				post.To = strings.TrimSuffix(strings.TrimPrefix(tokens[i+4], "#"), ".")
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

// 投稿に付けられたタグを取得
func (g *telegramTransferPostGateway) extractTags (message string, entities []tg.MessageEntityClass) []string {
	var tags []string
	if len(entities) == 0 {
		return tags
	}

	// UTF-16オフセットに対応するため、メッセージをruneのスライスに変換
	messageRunes := []rune(message)

	for _, entity := range entities {
		// エンティティがハッシュタグ型か判定
		if e, isHashtag := entity.(*tg.MessageEntityHashtag); isHashtag {
			// OffsetとLengthを使ってハッシュタグ部分を取得
			start := e.Offset
			end := e.Offset + e.Length
			if start >= 0 && end <= len(messageRunes) {
				tag := string(messageRunes[start:end])
				cleanTag := strings.TrimPrefix(tag, "#")
				tags = append(tags, cleanTag)
			}
		}
	}
	return tags
}