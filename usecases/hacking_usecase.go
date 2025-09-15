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

// ハッキング情報に関するユースケース
type HackingUsecase struct {
	repo             repository.HackingRepository
	telegramGateways []gateway.TelegramHackingPostGateway
	geminiGateway    gateway.GeminiGateway
	mu               sync.Mutex
}

// 新しいHackingUsecaseを生成
func NewHackingUsecase(repo repository.HackingRepository, telegramGateways []gateway.TelegramHackingPostGateway, geminiGateway gateway.GeminiGateway) *HackingUsecase {
	return &HackingUsecase{
		repo:             repo,
		telegramGateways: telegramGateways,
		geminiGateway:    geminiGateway,
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

func (uc *HackingUsecase) SetLastMessageIDToGateway(ctx context.Context) error {
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

func (uc *HackingUsecase) StoreLastMessageID(ctx context.Context) error {
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

// Telegramから100件以下の投稿を取得し、DBに保存
func (uc *HackingUsecase) ScrapeAndStore(ctx context.Context, limit int) (int, []error) {
	// 全ての新しい投稿を取得
	var wg sync.WaitGroup
	errsChan := make(chan error, len(uc.telegramGateways))
	var posts [][]*gateway.HackingPost

	for i, gw := range uc.telegramGateways {
		wg.Add(1)
		posts = append(posts, []*gateway.HackingPost{})
		go func(gw gateway.TelegramHackingPostGateway) {
			defer wg.Done()
			newPosts, err := gw.GetPosts(ctx, limit)
			if err != nil {
				errsChan <- fmt.Errorf("failed to get posts from telegram: %w", err)
				return
			}
			uc.mu.Lock()
			posts[i] = newPosts
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

	var allErrors []error
	var allProcessedCount int

	for i, gw := range uc.telegramGateways {
		if len(posts[i]) == 0 {
			continue
		}

		errsChan = make(chan error, len(posts[i]))
		messageIDChan := make(chan int, len(posts[i]))

		for _, post := range posts[i] {
			err := uc.processSinglePost(ctx, post)
			if err != nil {
				// エラーが発生したらチャネルに送信
				errsChan <- fmt.Errorf("failed to process post %s: %w", post.TxHash, err)
			} else {
				messageIDChan <- post.MessageID
			}
		}

		close(errsChan)
		close(messageIDChan)

		var errors []error
		for err := range errsChan {
			errors = append(errors, err)
		}

		for messageID := range messageIDChan {
			if messageID > gw.LastMessageID() {
				gw.SetLastMessageID(messageID)
			}
		}

		allErrors = append(allErrors, errors...)
		allProcessedCount = allProcessedCount + len(posts[i]) - len(errors)
	}

	log.Printf("Hacking Post: Scraping finished. Processed: %d, Errors: %d", allProcessedCount, len(allErrors))

	return allProcessedCount, allErrors
}

// Telegramから101件以上の投稿を取得し、DBに保存
func (uc *HackingUsecase) InitialScrapeAndStore(ctx context.Context, limit int) (int, []error) {
	// 全ての新しい投稿を取得
	var wg sync.WaitGroup
	errsChan := make(chan error, len(uc.telegramGateways))
	var posts [][]*gateway.HackingPost

	for i, gw := range uc.telegramGateways {
		wg.Add(1)
		posts = append(posts, []*gateway.HackingPost{})
		go func(gw gateway.TelegramHackingPostGateway) {
			defer wg.Done()
			newPosts, err := gw.GetPostsOver100(ctx, limit)
			if err != nil {
				errsChan <- fmt.Errorf("failed to get posts from telegram: %w", err)
				return
			}
			uc.mu.Lock()
			posts[i] = newPosts
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

	var allErrors []error
	var allProcessedCount int

	for i, gw := range uc.telegramGateways {
		if len(posts[i]) == 0 {
			continue
		}

		errsChan = make(chan error, len(posts[i]))
		messageIDChan := make(chan int, len(posts[i]))

		for _, post := range posts[i] {
			wg.Add(1)
			go func(p *gateway.HackingPost) {
				defer wg.Done()

				// 個別の投稿を処理するヘルパー関数
				err := uc.processSinglePost(ctx, p)
				if err != nil {
					// エラーが発生したらチャネルに送信
					errsChan <- fmt.Errorf("failed to process post from %s: %w", p.TxHash, err)
				} else {
					messageIDChan <- p.MessageID
				}
			}(post)
		}

		wg.Wait()
		close(errsChan)
		close(messageIDChan)

		var errors []error
		for err := range errsChan {
			errors = append(errors, err)
		}

		for messageID := range messageIDChan {
			if messageID > gw.LastMessageID() {
				gw.SetLastMessageID(messageID)
			}
		}

		allErrors = append(allErrors, errors...)
		allProcessedCount = allProcessedCount + len(posts[i]) - len(errors)
	}

	log.Printf("Scraping finished. Processed: %d, Errors: %d", allProcessedCount, len(allErrors))

	return allProcessedCount, allErrors
}

// 単一の投稿を処理するヘルパー関数
func (uc *HackingUsecase) processSinglePost(ctx context.Context, post *gateway.HackingPost) error {
	log.Printf("Processing post: %s", post.TxHash)

	// Geminiでテキストを分析
	extractedInfo, err := uc.geminiGateway.AnalyzeAndExtract(ctx, post)
	if err != nil {
		return fmt.Errorf("gemini analysis failed: %w", err)
	}

	infoToStore := &entity.HackingInfo{
		Protocol:   extractedInfo.Protocol,
		Network:    extractedInfo.Network,
		Amount:     extractedInfo.Amount,
		TxHash:     extractedInfo.TxHash,
		ReportTime: post.ReportTime,
		MessageID:  post.MessageID,
	}

	// DBに保存
	_, err = uc.repo.StoreInfo(ctx, infoToStore, extractedInfo.TagNames)
	if err != nil {
		return fmt.Errorf("database store failed: %w", err)
	}

	log.Printf("Successfully stored info: %s", infoToStore.TxHash)
	log.Printf("Tags: %s", extractedInfo.TagNames)
	return nil
}
