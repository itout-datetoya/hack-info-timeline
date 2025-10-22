package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/itout-datetoya/hack-info-timeline/domain/entity"

	"github.com/patrickmn/go-cache"
)

// HackingRepository インターフェースを実装する構造体
type transferRepository struct {
	dbRepo *dbTransferRepository
	cache *cache.Cache
}

// hackingRepository の新しいインスタンスを生成
func NewTransferRepository(dbRepo *dbTransferRepository, cache *cache.Cache) *transferRepository {
	return &transferRepository{dbRepo: dbRepo, cache: cache}
}

// 指定したタグ名に一致する情報を指定の件数取得
func (r *transferRepository) GetInfosByTagNames(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.TransferInfo, error) {

	return r.dbRepo.GetInfosByTagNames(ctx, tagNames, infoNumber)
}

// 指定したタグ名に一致する情報の内、指定した情報より過去から指定の件数取得
func (r *transferRepository) GetPrevInfosByTagNames(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.TransferInfo, error) {

	return r.dbRepo.GetPrevInfosByTagNames(ctx, tagNames, prevInfoID, infoNumber)
}

// 存在するすべてのタグを取得
func (r *transferRepository) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	var tags []*entity.Tag

    const key = "tags:all"

    if tags, found := r.cache.Get(key); found {
        if tags, ok := tags.([]*entity.Tag); ok {
            return tags, nil
        }
    }
	
	tags, err := r.dbRepo.GetAllTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	r.cache.Set(key, tags, 15*time.Minute)

	return tags, nil
}

func (r *transferRepository) SetTagToCache(ctx context.Context) error {
	const key = "tags:all"

	tags, err := r.dbRepo.GetAllTags(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	r.cache.Set(key, tags, 15*time.Minute)

	return nil
}

// 新しいハッキング情報と関連タグをトランザクション内で保存
func (r *transferRepository) StoreInfo(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error) {

	return r.dbRepo.StoreInfo(ctx, info, tagNames)
}

// チャンネル情報をトランザクション内で保存
func (r *transferRepository) StoreChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error {

	return r.dbRepo.StoreChannelStatus(ctx, channelStatus)
}

// チャンネル情報をトランザクション内で更新
func (r *transferRepository) UpdateChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error {

	return r.dbRepo.UpdateChannelStatus(ctx, channelStatus)
}

// usernameで指定されたチャンネル情報を1件取得
func (r *transferRepository) GetChannelStatusByUsername(ctx context.Context, username string) (*entity.TelegramChannel, error) {

	return r.dbRepo.GetChannelStatusByUsername(ctx, username)
}
