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
	mu               sync.Mutex
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

// Telegramから投稿を取得し、DBに保存
func (uc *TransferUsecase) ScrapeAndStore(ctx context.Context, limit int) (int, []error) {
	// 全ての新しい投稿を取得
	var wg sync.WaitGroup
	errsChan := make(chan error, len(uc.telegramGateways))
	var posts []*gateway.TransferPost

	for _, gw := range uc.telegramGateways {
		wg.Add(1)
		go func(gw gateway.TelegramTransferPostGateway) {
			defer wg.Done()
			newPosts, err := gw.GetPosts(ctx, limit)
			if err != nil {
				errsChan <- fmt.Errorf("failed to get posts from telegram: %w", err)
				return
			}
			uc.mu.Lock()
			posts = append(posts, newPosts...)
			uc.mu.Unlock()
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

	if len(posts) == 0 {
		log.Println("Info: No new messages to process.")
		return 0, nil
	}

	// 各投稿を並行処理
	errsChan = make(chan error, len(posts))

	for _, post := range posts {
		wg.Add(1)
		go func(p *gateway.TransferPost) {
			defer wg.Done()

			// 個別の投稿を処理するヘルパー関数
			err := uc.processSinglePost(ctx, p)
			if err != nil {
				// エラーが発生したらチャンネルに送信
				errsChan <- fmt.Errorf("failed to process post: %w", err)
			}
		}(post)
	}

	wg.Wait()
	close(errsChan)

	var allErrors []error
	for err := range errsChan {
		allErrors = append(allErrors, err)
	}

	processedCount := len(posts) - len(allErrors)
	log.Printf("Scraping finished. Processed: %d, Errors: %d", processedCount, len(allErrors))

	return processedCount, allErrors
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
