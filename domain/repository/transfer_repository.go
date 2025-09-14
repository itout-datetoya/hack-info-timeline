package repository

import (
	"context"
	"github.com/itout-datetoya/hack-info-timeline/domain/entity"
)

// 送金情報の永続化
type TransferRepository interface {
	// 指定したタグ名に一致する送金情報を最新から指定の件数取得
	GetInfosByTagNames(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.TransferInfo, error)

	// 指定したタグ名に一致する送金情報の内、指定した情報より過去から指定の件数取得
	GetPrevInfosByTagNames(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.TransferInfo, error)

	// 存在するすべてのタグを出力
	GetAllTags(ctx context.Context) ([]*entity.Tag, error)

	// 新しい送金情報をトランザクション内で保存
	// 新しいタグの保存と、中間テーブルへの関連付けも実行
	StoreInfo(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error)

	// チャンネル情報を保存
	StoreChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error
	// チャンネル情報を更新
	UpdateChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error
	// チャンネル情報を取得
	GetChannelStatusByUsername(ctx context.Context, username string) (*entity.TelegramChannel, error)
}
