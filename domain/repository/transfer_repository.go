package repository

import (
	"context"
	"github.com/itout-datetoya/hack-info-timeline/domain/entity"
)

// 送金情報の永続化
type TransferRepository interface {
	// 指定されたタグ名に一致する送金情報を検索
	FindByTagNames(ctx context.Context, tagNames []string) ([]*entity.TransferInfo, error)
	
	// 存在するすべてのタグを出力
	ListTags(ctx context.Context) ([]*entity.Tag, error)
	
	// 新しい送金情報をトランザクション内で保存
	// 新しいタグの保存と、中間テーブルへの関連付けも実行
	Store(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error)
}