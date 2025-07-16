package gateway

import (
	"context"
	"errors"
	"fmt"
	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/tg"
)

// TelegramHackingPostGatewayを実装する構造体
type telegramHackingPostGateway struct {
	manager         *TelegramClientManager
	channelUsername string
	channel         *tg.Channel
	lastMessageID   int
	mu              sync.Mutex
}

// 新しいtelegramHackingPostGatewayを生成
func NewTelegramHackingPostGateway(manager *TelegramClientManager, channelUsername string) gateway.TelegramHackingPostGateway {
	return &telegramHackingPostGateway{manager: manager, channelUsername: channelUsername}
}

// 最後に取得した投稿以降、最新の投稿を取得
func (g *telegramHackingPostGateway) GetPosts(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
	api := g.manager.API()
	if api == nil {
		return nil, errors.New("telegram client is not ready")
	}

	// チャンネル名を解決してPeer情報を取得
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: g.channelUsername,
	})
	if err != nil {
		return nil, fmt.Errorf("gateway A: failed to resolve username %s: %w", g.channelUsername, err)
	}
	channel, ok := resolved.Chats[0].(*tg.Channel)
	if !ok {
		return nil, fmt.Errorf("gateway A: resolved peer is not a channel")
	}
	g.channel = channel
	inputPeer := channel.AsInputPeer()

	g.mu.Lock()
	defer g.mu.Unlock()
	// 最後に取得した投稿以降、最新の投稿を取得
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  inputPeer,
		MinID: g.lastMessageID,
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("gateway A: failed to get channel history: %w", err)
	}

	// 取得した投稿の内、ハッキング情報を含むものをHackingPostに変換
	return g.convertMessages(ctx, history)
}

// 取得した投稿の内、ハッキング情報を含むものをHackingPostに変換
// 関連ポストを追加で取得
func (g *telegramHackingPostGateway) convertMessages(ctx context.Context, history tg.MessagesMessagesClass) ([]*gateway.HackingPost, error) {
	// 取得したデータを投稿のスライスに変換
	channelMessages, ok := history.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, fmt.Errorf("gateway A: failed to cast history to ChannelMessages")
	}

	var posts []*gateway.HackingPost
	for _, msg := range channelMessages.Messages {
		// チャンネルの投稿か確認
		if message, ok := msg.(*tg.Message); ok && message.Message != "" {
			// リプライ先があるか確認
			if replyTo, ok := message.GetReplyTo(); ok {
				// リプライ先IDが取得可能か確認
				if messageReplyTo, ok := replyTo.(*tg.MessageReplyHeader); ok {
					// リプライ先IDを取得
					replyToID, _ := messageReplyTo.GetReplyToMsgID()
					id := []tg.InputMessageClass{&tg.InputMessageID{ID: replyToID}}
					// リプライ先の投稿を取得
					inputChannel := g.channel.AsInput()
					repliedMsgs, err := g.manager.api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
						Channel: inputChannel,
						ID:      id,
					})
					if err != nil {
						return nil, fmt.Errorf("gateway A: failed to get replied message: %w", err)
					}
					repliedMessages, _ := repliedMsgs.(*tg.MessagesChannelMessages)
					repliedMessage, _ := repliedMessages.Messages[0].(*tg.Message)

					// リプライ先にさらにリプライがあればその投稿は使用しない
					if _, ok := repliedMessage.GetReplyTo(); !ok {
						// リプライ先からハッキング情報を取得
						post, err := g.parseHackingMessage(repliedMessage.Message)
						if err == nil {
							// 投稿内容を添付
							date := repliedMessage.GetDate()
							post.ReportTime = time.Unix(int64(date), 0)
							post.Text = message.Message
							posts = append(posts, post)
						} else {
							log.Printf("Failed to parse hacking message with error: %v", err)
						}
					}
				}
			}
			// 最後に取得した投稿のIDを更新
			if message.ID > g.lastMessageID {
				g.lastMessageID = message.ID
			}
		}
	}
	return posts, nil
}

// 投稿の形式からパースしてハッキング情報を取得
func (g *telegramHackingPostGateway) parseHackingMessage(message string) (*gateway.HackingPost, error) {
	// スペースで分割
	tokens := strings.Fields(message)

	found := false
	var post gateway.HackingPost

	// "Network:", "Exploit:", "Balance" を基準にパース
	for i, token := range tokens {
		if token == "Network:" && i < len(tokens) {
			// "Network:" の次の単語が「ネットワーク」
			post.Network = tokens[i+1]
			continue
		}
		if token == "Exploit:" && i < len(tokens) {
			// "Exploit:" の次の単語が「TX Hash」
			post.TxHash = tokens[i+1]
			continue
		}
		if token == "Balance" && i+1 < len(tokens) {
			// "Balance" の2つ先の単語が「送金額」
			post.Amount = tokens[i+2]
			found = true
			break
		}
	}

	if !found {
		return nil, errors.New("HackingPost pattern not found in message")
	}

	return &post, nil
}
