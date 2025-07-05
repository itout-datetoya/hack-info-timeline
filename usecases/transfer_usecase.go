package usecases

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"github.com/itout-datetoya/hack-info-timeline/domain/entity"
	"github.com/itout-datetoya/hack-info-timeline/domain/repository"
	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"
)

// 送金情報に関するユースケース
type TransferUsecase struct {
	repo repository.TransferRepository
	telegramGateway gateway.TelegramTransferPostGateway
}

// 新しいTransferUsecaseを生成
func NewTransferUsecase(repo repository.TransferRepository, telegramGateway gateway.TelegramTransferPostGateway) *TransferUsecase {
	return &TransferUsecase{
		repo: repo,
		telegramGateway: telegramGateway,
	}
}

// タイムライン情報を取得
func (uc *TransferUsecase) GetTimeline(ctx context.Context, tagNames []string) ([]*entity.TransferInfo, error) {
	return uc.repo.FindByTagNames(ctx, tagNames)
}

// 全てのタグを取得
func (uc *TransferUsecase) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	return uc.repo.ListTags(ctx)
}

// Telegramから投稿を取得し、DBに保存
func (uc *TransferUsecase) ScrapeAndStore(ctx context.Context) (int, []error) {
	// 全ての新しい投稿を取得
	posts, err := uc.telegramGateway.GetPosts(ctx)
	if err != nil {
		return 0, []error{fmt.Errorf("failed to get posts from telegram: %w", err)}
	}

	if len(posts) == 0 {
		log.Println("Info: No new messages to process.")
		return 0, nil
	}

	// 各投稿を並行処理
	var wg sync.WaitGroup
	errsChan := make(chan error, len(posts))

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
	Token:		post.Token,
	Amount:		post.Amount,
	From:		post.From,
	To:			post.To,
	ReportTime:	time.Now(),
	}

	// DBに保存
	_, err := uc.repo.Store(ctx, infoToStore, post.TagNames)
	if err != nil {
		return fmt.Errorf("database store failed: %w", err)
	}

	log.Printf("Successfully stored %s %s Transfer", post.Amount, post.Token)
	return nil
}