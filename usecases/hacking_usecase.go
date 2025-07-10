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

// ハッキング情報に関するユースケース
type HackingUsecase struct {
	repo repository.HackingRepository
	telegramGateway gateway.TelegramHackingPostGateway
	geminiGateway gateway.GeminiGateway
}

// 新しいHackingUsecaseを生成
func NewHackingUsecase(repo repository.HackingRepository, telegramGateway gateway.TelegramHackingPostGateway, geminiGateway gateway.GeminiGateway) *HackingUsecase {
	return &HackingUsecase{
		repo: repo,
		telegramGateway: telegramGateway,
		geminiGateway: geminiGateway,
	}
}

// 最新タイムライン情報を指定件数取得
func (uc *HackingUsecase) GetLatestTimeline(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.HackingInfo, error) {
	return uc.repo.GetInfosByTagNames(ctx, tagNames, infoNumber)
}

// 指定情報より過去のタイムライン情報を指定件数取得
func (uc *HackingUsecase) GetPrevTimeline(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.HackingInfo, error) {
	return uc.repo.GetPrevInfosByTagNames(ctx, tagNames, prevInfoID, infoNumber)
}

// 全てのタグを取得
func (uc *HackingUsecase) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	return uc.repo.GetAllTags(ctx)
}

// Telegramから投稿を取得し、DBに保存
func (uc *HackingUsecase) ScrapeAndStore(ctx context.Context) (int, []error) {
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
		go func(p *gateway.HackingPost) {
			defer wg.Done()

			// 個別の投稿を処理するヘルパー関数
			err := uc.processSinglePost(ctx, p)
			if err != nil {
				// エラーが発生したらチャンネルに送信
				errsChan <- fmt.Errorf("failed to process post from %s: %w", p.Text, err)
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
func (uc *HackingUsecase) processSinglePost(ctx context.Context, post *gateway.HackingPost) error {
	log.Printf("Processing post: %s", post.Text)

	// Geminiでテキストを分析
	extractedInfo, err := uc.geminiGateway.AnalyzeAndExtract(ctx, post)
	if err != nil {
		return fmt.Errorf("gemini analysis failed: %w", err)
	}

	infoToStore := &entity.HackingInfo{
	Protocol:	extractedInfo.Protocol,
	Network:	extractedInfo.Network,
	Amount:		extractedInfo.Amount,
	TxHash:		extractedInfo.TxHash,
	ReportTime:	time.Now(),
	}


	// DBに保存
	_, err = uc.repo.StoreInfo(ctx, infoToStore, extractedInfo.TagNames)
	if err != nil {
		return fmt.Errorf("database store failed: %w", err)
	}

	log.Printf("Successfully stored info: %s", infoToStore.TxHash)
	return nil
}