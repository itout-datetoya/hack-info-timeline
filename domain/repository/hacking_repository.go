package repository

import (
	"context"
	"github.com/itout-datetoya/hack-info-timeline/domain/entity"
)

// ハッキング情報の永続化
type HackingRepository interface {
	// 指定されたタグ名に一致するハッキング情報を検索
	FindByTagNames(ctx context.Context, tagNames []string) ([]*entity.HackingInfo, error)
	
	// 存在するすべてのタグを出力
	ListTags(ctx context.Context) ([]*entity.Tag, error)
	
	// 新しいハッキング情報をトランザクション内で保存
	// 新しいタグの保存と、中間テーブルへの関連付けも実行
	Store(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error)
}