package repository

import (
	"context"
	"github.com/itout-datetoya/hack-info-timeline/domain/entity"
)

// ハッキング情報の永続化
type HackingRepository interface {
	// 指定したタグ名に一致するハッキング情報を最新から指定の件数取得
	GetInfosByTagNames(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.HackingInfo, error)

	// 指定したタグ名に一致するハッキング情報の内、指定した情報より過去から指定の件数取得
	GetPrevInfosByTagNames(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.HackingInfo, error)
	
	// 存在するすべてのタグを出力
	GetAllTags(ctx context.Context) ([]*entity.Tag, error)
	
	// 新しいハッキング情報をトランザクション内で保存
	// 新しいタグの保存と、中間テーブルへの関連付けも実行
	StoreInfo(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error)
}