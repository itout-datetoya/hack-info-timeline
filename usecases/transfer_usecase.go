package usecases

import (
	"context"
	"fmt"
	"github.com/itout-datetoya/hack-info-timeline/domain/entity"
	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"
	"github.com/itout-datetoya/hack-info-timeline/domain/repository"
	"log"
	"sync"
)

// 送金情報に関するユースケース
type TransferUsecase struct {
	repo             repository.TransferRepository
	telegramGateways []gateway.TelegramTransferPostGateway
}

// 新しいTransferUsecaseを生成
func NewTransferUsecase(repo repository.TransferRepository, telegramGateways []gateway.TelegramTransferPostGateway) *TransferUsecase {
	return &TransferUsecase{
		repo:             repo,
		telegramGateways: telegramGateways,
	}
}

// 最新タイムライン情報を指定件数取得
func (uc *TransferUsecase) GetLatestTimeline(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.TransferInfo, error) {
	return uc.repo.GetInfosByTagNames(ctx, tagNames, infoNumber)
}

// 指定情報より過去のタイムライン情報を指定件数取得
func (uc *TransferUsecase) GetPrevTimeline(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.TransferInfo, error) {
	return uc.repo.GetPrevInfosByTagNames(ctx, tagNames, prevInfoID, infoNumber)
}

// 全てのタグを取得
func (uc *TransferUsecase) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	return uc.repo.GetAllTags(ctx)
}

// DBからタグを取得してキャッシュに保存
func (uc *TransferUsecase) SetTagToCache(ctx context.Context) error {
	return uc.repo.SetTagToCache(ctx)
}

func (uc *TransferUsecase) SetLastMessageIDToGateway(ctx context.Context) error {
	for _, gw := range uc.telegramGateways {
		channelStatus, err := uc.repo.GetChannelStatusByUsername(ctx, gw.ChannelUsername())
		if err != nil {
			return fmt.Errorf("failed to get channel status: %w", err)
		}
		if channelStatus == nil {
			newChannelStatus := entity.TelegramChannel{ChannelUsername: gw.ChannelUsername(), LastMessageID: 0}
			err = uc.repo.StoreChannelStatus(ctx, &newChannelStatus)
			if err != nil {
				return fmt.Errorf("failed to store channel status: %w", err)
			}
		} else {
			gw.SetLastMessageID(channelStatus.LastMessageID)
		}
	}

	return nil
}

func (uc *TransferUsecase) StoreLastMessageID(ctx context.Context) error {
	for _, gw := range uc.telegramGateways {
		newChannelStatus := entity.TelegramChannel{ChannelUsername: gw.ChannelUsername(), LastMessageID: gw.LastMessageID()}
		channelStatus, err := uc.repo.GetChannelStatusByUsername(ctx, gw.ChannelUsername())
		if err != nil {
			return fmt.Errorf("failed to get channel status: %w", err)
		}
		if channelStatus == nil {
			err = uc.repo.StoreChannelStatus(ctx, &newChannelStatus)
			if err != nil {
				return fmt.Errorf("failed to store channel status: %w", err)
			}
		} else {
			err = uc.repo.UpdateChannelStatus(ctx, &newChannelStatus)
			if err != nil {
				return fmt.Errorf("failed to update channel status: %w", err)
			}
		}
	}

	return nil
}

// Telegramから投稿を取得し、DBに保存
func (uc *TransferUsecase) ScrapeAndStore(ctx context.Context, limit int) (int, []error) {
	// 全ての新しい投稿を取得
	var wg sync.WaitGroup
	errsChan := make(chan error, len(uc.telegramGateways))
	var posts [][]*gateway.TransferPost

	for i, gw := range uc.telegramGateways {
		wg.Add(1)
		posts = append(posts, []*gateway.TransferPost{})
		go func(gw gateway.TelegramTransferPostGateway) {
			defer wg.Done()
			newPosts, err := gw.GetPosts(ctx, limit)
			if err != nil {
				errsChan <- fmt.Errorf("failed to get posts from telegram: %w", err)
				return
			}
			posts[i] = newPosts
		}(gw)
	}

	wg.Wait()
	close(errsChan)

	var getPostsErrors []error
	for err := range errsChan {
		getPostsErrors = append(getPostsErrors, err)
	}

	if len(getPostsErrors) != 0 {
		return 0, getPostsErrors
	}

	var allErrors []error
	var allProcessedCount int

	for i := range uc.telegramGateways {
		if len(posts[i]) == 0 {
			continue
		}

		errsChan = make(chan error, len(posts[i]))

		for _, post := range posts[i] {
			wg.Add(1)
			go func(p *gateway.TransferPost) {
				defer wg.Done()

				// 個別の投稿を処理するヘルパー関数
				err := uc.processSinglePost(ctx, p)
				if err != nil {
					// エラーが発生したらチャネルに送信
					errsChan <- fmt.Errorf("failed to process post %s %s Transfer: %w", p.Amount, p.Token, err)
				}
			}(post)
		}

		wg.Wait()
		close(errsChan)

		var errors []error
		for err := range errsChan {
			errors = append(errors, err)
		}

		allErrors = append(allErrors, errors...)
		allProcessedCount = allProcessedCount + len(posts[i]) - len(errors)
	}

	log.Printf("Transfer Post: Scraping finished. Processed: %d, Errors: %d", allProcessedCount, len(allErrors))

	return allProcessedCount, allErrors
}

// 単一の投稿を処理するヘルパー関数
func (uc *TransferUsecase) processSinglePost(ctx context.Context, post *gateway.TransferPost) error {
	log.Printf("Processing post: %s %s Transfer", post.Amount, post.Token)

	infoToStore := &entity.TransferInfo{
		Token:      post.Token,
		Amount:     post.Amount,
		From:       post.From,
		To:         post.To,
		ReportTime: post.ReportTime,
		MessageID:  post.MessageID,
	}

	// DBに保存
	_, err := uc.repo.StoreInfo(ctx, infoToStore, post.TagNames)
	if err != nil {
		return fmt.Errorf("database store failed: %w", err)
	}

	log.Printf("Successfully stored %s %s Transfer", post.Amount, post.Token)
	log.Printf("Tags: %s", post.TagNames)
	return nil
}
