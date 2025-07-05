package gateway

import (
	"context"
	"errors"
	"fmt"
	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"

	"github.com/gotd/td/tg"
)

// TransferHackingPostGatewayを実装する構造体
type telegramTransferPostGateway struct {
	manager			*TelegramClientManager
	channelUsername	string
	channelPeer     *tg.InputPeerChannel
	lastMessageID	int
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
		return nil, fmt.Errorf("gateway A: failed to resolve username %s: %w", g.channelUsername, err)
	}
	channel, ok := resolved.Chats[0].(*tg.Channel)
	if !ok {
		return nil, fmt.Errorf("gateway A: resolved peer is not a channel")
	}
	inputPeer := channel.AsInputPeer()
	g.channelPeer = inputPeer

	// 最後に取得した投稿以降、最新100件の投稿を取得
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  inputPeer,
		MinID: g.lastMessageID, 
		Limit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("gateway A: failed to get channel history: %w", err)
	}

	// 取得した投稿の内、送金情報を含むものをTransferPostに変換
	return g.convertMessages(history)
}

// 取得した投稿の内、送金情報を含むものをHackingPostに変換
func (g *telegramTransferPostGateway) convertMessages(history tg.MessagesMessagesClass) ([]*gateway.TransferPost, error) {
	// 取得したデータを投稿のスライスに変換
	channelMessages, ok := history.(*tg.MessagesChannelMessages)
	if !ok {
		return nil, fmt.Errorf("gateway A: failed to cast history to ChannelMessages")
	}

	var posts []*gateway.TransferPost
	for _, msg := range channelMessages.Messages {
		// チャンネルの投稿か確認
		if message, ok := msg.(*tg.Message); ok && message.Message != "" {
			
			// ToDo parse message
			
			posts = append(posts, &gateway.TransferPost{
				Token:	"",
				Amount:	"",
				From:	"",
				To:		"",
			})
		}
	}
	return posts, nil
}