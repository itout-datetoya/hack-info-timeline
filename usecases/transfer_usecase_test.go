package usecases

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/itout-datetoya/hack-info-timeline/domain/entity"
	"github.com/itout-datetoya/hack-info-timeline/domain/gateway"
)

// ==================== Mock Implementations ====================

// mockTransferRepository は TransferRepository インターフェースのモック実装
type mockTransferRepository struct {
	getInfosByTagNamesFunc         func(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.TransferInfo, error)
	getPrevInfosByTagNamesFunc     func(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.TransferInfo, error)
	getAllTagsFunc                 func(ctx context.Context) ([]*entity.Tag, error)
	setTagToCacheFunc              func(ctx context.Context) error
	storeInfoFunc                  func(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error)
	storeChannelStatusFunc         func(ctx context.Context, channelStatus *entity.TelegramChannel) error
	updateChannelStatusFunc        func(ctx context.Context, channelStatus *entity.TelegramChannel) error
	getChannelStatusByUsernameFunc func(ctx context.Context, username string) (*entity.TelegramChannel, error)
}

func (m *mockTransferRepository) GetInfosByTagNames(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.TransferInfo, error) {
	if m.getInfosByTagNamesFunc != nil {
		return m.getInfosByTagNamesFunc(ctx, tagNames, infoNumber)
	}
	return nil, nil
}

func (m *mockTransferRepository) GetPrevInfosByTagNames(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.TransferInfo, error) {
	if m.getPrevInfosByTagNamesFunc != nil {
		return m.getPrevInfosByTagNamesFunc(ctx, tagNames, prevInfoID, infoNumber)
	}
	return nil, nil
}

func (m *mockTransferRepository) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	if m.getAllTagsFunc != nil {
		return m.getAllTagsFunc(ctx)
	}
	return nil, nil
}

func (m *mockTransferRepository) SetTagToCache(ctx context.Context) error {
	if m.setTagToCacheFunc != nil {
		return m.setTagToCacheFunc(ctx)
	}
	return nil
}

func (m *mockTransferRepository) StoreInfo(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error) {
	if m.storeInfoFunc != nil {
		return m.storeInfoFunc(ctx, info, tagNames)
	}
	return 0, nil
}

func (m *mockTransferRepository) StoreChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error {
	if m.storeChannelStatusFunc != nil {
		return m.storeChannelStatusFunc(ctx, channelStatus)
	}
	return nil
}

func (m *mockTransferRepository) UpdateChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error {
	if m.updateChannelStatusFunc != nil {
		return m.updateChannelStatusFunc(ctx, channelStatus)
	}
	return nil
}

func (m *mockTransferRepository) GetChannelStatusByUsername(ctx context.Context, username string) (*entity.TelegramChannel, error) {
	if m.getChannelStatusByUsernameFunc != nil {
		return m.getChannelStatusByUsernameFunc(ctx, username)
	}
	return nil, nil
}

// mockTelegramTransferPostGateway は TelegramTransferPostGateway インターフェースのモック実装
type mockTelegramTransferPostGateway struct {
	channelUsername string
	lastMessageID   int
	getPostsFunc    func(ctx context.Context, limit int) ([]*gateway.TransferPost, error)
	mu              sync.Mutex
}

func (m *mockTelegramTransferPostGateway) SetLastMessageID(lastMessageID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastMessageID = lastMessageID
}

func (m *mockTelegramTransferPostGateway) LastMessageID() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastMessageID
}

func (m *mockTelegramTransferPostGateway) ChannelUsername() string {
	return m.channelUsername
}

func (m *mockTelegramTransferPostGateway) GetPosts(ctx context.Context, limit int) ([]*gateway.TransferPost, error) {
	if m.getPostsFunc != nil {
		return m.getPostsFunc(ctx, limit)
	}
	return nil, nil
}

// ==================== Test Helper Functions ====================

func createTestTransferPost(messageID int, token, amount string) *gateway.TransferPost {
	return &gateway.TransferPost{
		Token:      token,
		Amount:     amount,
		From:       "0xSender123",
		To:         "0xRecipient456",
		ReportTime: time.Now(),
		MessageID:  messageID,
		TagNames:   []string{"Transfer", "Bridge"},
	}
}

func createTestTransferInfo(id int64, token, amount string) *entity.TransferInfo {
	return &entity.TransferInfo{
		ID:         id,
		Token:      token,
		Amount:     amount,
		From:       "0xSender123",
		To:         "0xRecipient456",
		ReportTime: time.Now(),
		MessageID:  100,
		Tags:       []*entity.Tag{{ID: 1, Name: "Transfer"}},
	}
}

// ==================== Simple Method Tests ====================

func TestTransferGetLatestTimeline(t *testing.T) {
	tests := []struct {
		name       string
		tagNames   []string
		infoNumber int
		mockResult []*entity.TransferInfo
		mockError  error
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "success case",
			tagNames:   []string{"Transfer", "Bridge"},
			infoNumber: 10,
			mockResult: []*entity.TransferInfo{
				createTestTransferInfo(1, "USDC", "1000000"),
				createTestTransferInfo(2, "ETH", "500000"),
			},
			mockError: nil,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:       "empty result",
			tagNames:   []string{"NonExistent"},
			infoNumber: 10,
			mockResult: []*entity.TransferInfo{},
			mockError:  nil,
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "repository error",
			tagNames:   []string{"Transfer"},
			infoNumber: 10,
			mockResult: nil,
			mockError:  errors.New("database error"),
			wantErr:    true,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockTransferRepository{
				getInfosByTagNamesFunc: func(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.TransferInfo, error) {
					return tt.mockResult, tt.mockError
				},
			}

			uc := NewTransferUsecase(mockRepo, nil)
			ctx := context.Background()

			result, err := uc.GetLatestTimeline(ctx, tt.tagNames, tt.infoNumber)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestTimeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantCount {
				t.Errorf("GetLatestTimeline() returned %d items, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestTransferGetPrevTimeline(t *testing.T) {
	tests := []struct {
		name       string
		tagNames   []string
		prevInfoID int64
		infoNumber int
		mockResult []*entity.TransferInfo
		mockError  error
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "success case",
			tagNames:   []string{"Transfer"},
			prevInfoID: 100,
			infoNumber: 5,
			mockResult: []*entity.TransferInfo{
				createTestTransferInfo(99, "USDC", "1000000"),
				createTestTransferInfo(98, "ETH", "500000"),
			},
			mockError: nil,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:       "no previous data",
			tagNames:   []string{"Transfer"},
			prevInfoID: 1,
			infoNumber: 5,
			mockResult: []*entity.TransferInfo{},
			mockError:  nil,
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "repository error",
			tagNames:   []string{"Transfer"},
			prevInfoID: 100,
			infoNumber: 5,
			mockResult: nil,
			mockError:  errors.New("database error"),
			wantErr:    true,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockTransferRepository{
				getPrevInfosByTagNamesFunc: func(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.TransferInfo, error) {
					return tt.mockResult, tt.mockError
				},
			}

			uc := NewTransferUsecase(mockRepo, nil)
			ctx := context.Background()

			result, err := uc.GetPrevTimeline(ctx, tt.tagNames, tt.prevInfoID, tt.infoNumber)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPrevTimeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantCount {
				t.Errorf("GetPrevTimeline() returned %d items, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestTransferGetAllTags(t *testing.T) {
	tests := []struct {
		name       string
		mockResult []*entity.Tag
		mockError  error
		wantErr    bool
		wantCount  int
	}{
		{
			name: "success case",
			mockResult: []*entity.Tag{
				{ID: 1, Name: "Transfer"},
				{ID: 2, Name: "Bridge"},
				{ID: 3, Name: "Swap"},
			},
			mockError: nil,
			wantErr:   false,
			wantCount: 3,
		},
		{
			name:       "empty tags",
			mockResult: []*entity.Tag{},
			mockError:  nil,
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "repository error",
			mockResult: nil,
			mockError:  errors.New("database error"),
			wantErr:    true,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockTransferRepository{
				getAllTagsFunc: func(ctx context.Context) ([]*entity.Tag, error) {
					return tt.mockResult, tt.mockError
				},
			}

			uc := NewTransferUsecase(mockRepo, nil)
			ctx := context.Background()

			result, err := uc.GetAllTags(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllTags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantCount {
				t.Errorf("GetAllTags() returned %d items, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestTransferSetTagToCache(t *testing.T) {
	tests := []struct {
		name      string
		mockError error
		wantErr   bool
	}{
		{
			name:      "success case",
			mockError: nil,
			wantErr:   false,
		},
		{
			name:      "repository error",
			mockError: errors.New("cache error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockTransferRepository{
				setTagToCacheFunc: func(ctx context.Context) error {
					return tt.mockError
				},
			}

			uc := NewTransferUsecase(mockRepo, nil)
			ctx := context.Background()

			err := uc.SetTagToCache(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetTagToCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ==================== Process Single Post Tests ====================

func TestTransferProcessSinglePost(t *testing.T) {
	tests := []struct {
		name            string
		post            *gateway.TransferPost
		storeError      error
		wantErr         bool
		wantErrContains string
	}{
		{
			name:            "success case",
			post:            createTestTransferPost(100, "USDC", "1000000"),
			storeError:      nil,
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name:            "database store error",
			post:            createTestTransferPost(100, "USDC", "1000000"),
			storeError:      errors.New("constraint violation"),
			wantErr:         true,
			wantErrContains: "database store failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockTransferRepository{
				storeInfoFunc: func(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error) {
					if tt.storeError != nil {
						return 0, tt.storeError
					}
					return 1, nil
				},
			}

			uc := NewTransferUsecase(mockRepo, nil)
			ctx := context.Background()

			err := uc.processSinglePost(ctx, tt.post)

			if (err != nil) != tt.wantErr {
				t.Errorf("processSinglePost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrContains != "" {
				if err == nil || !contains(err.Error(), tt.wantErrContains) {
					t.Errorf("processSinglePost() error = %v, want error containing %q", err, tt.wantErrContains)
				}
			}
		})
	}
}

// ==================== ScrapeAndStore Tests ====================

func TestTransferScrapeAndStore(t *testing.T) {
	tests := []struct {
		name               string
		limit              int
		numGateways        int
		postsPerGateway    [][]*gateway.TransferPost
		getPostsErrors     []error
		processErrors      map[string]error // token+amount -> error
		wantProcessedCount int
		wantErrorCount     int
	}{
		{
			name:        "success case with single gateway",
			limit:       10,
			numGateways: 1,
			postsPerGateway: [][]*gateway.TransferPost{
				{
					createTestTransferPost(101, "USDC", "1000000"),
					createTestTransferPost(102, "ETH", "500000"),
					createTestTransferPost(103, "DAI", "2000000"),
				},
			},
			getPostsErrors:     []error{nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 3,
			wantErrorCount:     0,
		},
		{
			name:        "success with multiple gateways",
			limit:       10,
			numGateways: 2,
			postsPerGateway: [][]*gateway.TransferPost{
				{
					createTestTransferPost(101, "USDC", "1000000"),
					createTestTransferPost(102, "ETH", "500000"),
				},
				{
					createTestTransferPost(201, "DAI", "2000000"),
					createTestTransferPost(202, "USDT", "1500000"),
					createTestTransferPost(203, "WBTC", "100000"),
				},
			},
			getPostsErrors:     []error{nil, nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 5,
			wantErrorCount:     0,
		},
		{
			name:        "partial processing errors",
			limit:       10,
			numGateways: 1,
			postsPerGateway: [][]*gateway.TransferPost{
				{
					createTestTransferPost(101, "USDC", "1000000"),
					createTestTransferPost(102, "ETH", "500000"),
					createTestTransferPost(103, "DAI", "2000000"),
				},
			},
			getPostsErrors: []error{nil},
			processErrors: map[string]error{
				"ETH500000": errors.New("store error"),
			},
			wantProcessedCount: 2,
			wantErrorCount:     1,
		},
		{
			name:        "get posts error",
			limit:       10,
			numGateways: 2,
			postsPerGateway: [][]*gateway.TransferPost{
				nil,
				nil,
			},
			getPostsErrors:     []error{errors.New("telegram error"), nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 0,
			wantErrorCount:     1,
		},
		{
			name:               "no posts to process",
			limit:              10,
			numGateways:        1,
			postsPerGateway:    [][]*gateway.TransferPost{{}},
			getPostsErrors:     []error{nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 0,
			wantErrorCount:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockTransferRepository{
				storeInfoFunc: func(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error) {
					key := info.Token + info.Amount
					if err, exists := tt.processErrors[key]; exists {
						return 0, err
					}
					return 1, nil
				},
			}

			var gateways []gateway.TelegramTransferPostGateway
			for i := 0; i < tt.numGateways; i++ {
				idx := i
				mockGW := &mockTelegramTransferPostGateway{
					channelUsername: "channel" + string(rune('1'+i)),
					getPostsFunc: func(ctx context.Context, limit int) ([]*gateway.TransferPost, error) {
						if tt.getPostsErrors[idx] != nil {
							return nil, tt.getPostsErrors[idx]
						}
						return tt.postsPerGateway[idx], nil
					},
				}
				gateways = append(gateways, mockGW)
			}

			uc := NewTransferUsecase(mockRepo, gateways)
			ctx := context.Background()

			processedCount, errs := uc.ScrapeAndStore(ctx, tt.limit)

			if processedCount != tt.wantProcessedCount {
				t.Errorf("ScrapeAndStore() processedCount = %d, want %d", processedCount, tt.wantProcessedCount)
			}

			if len(errs) != tt.wantErrorCount {
				t.Errorf("ScrapeAndStore() errorCount = %d, want %d", len(errs), tt.wantErrorCount)
			}

			// LastMessageIDの更新責務はGatewayへ移動したため、
			// Usecase側での検証は行いません。
		})
	}
}

// Usecase側でのLastMessageID更新テストは責務変更により削除しました。

// ==================== Concurrency Tests ====================

func TestTransferScrapeAndStore_Concurrency(t *testing.T) {
	t.Run("concurrent gateway and post processing", func(t *testing.T) {
		numGateways := 5
		postsPerGateway := 5

		mockRepo := &mockTransferRepository{
			storeInfoFunc: func(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error) {
				// Simulate some processing time
				time.Sleep(1 * time.Millisecond)
				return 1, nil
			},
		}

		var gateways []gateway.TelegramTransferPostGateway
		for i := 0; i < numGateways; i++ {
			mockGW := &mockTelegramTransferPostGateway{
				channelUsername: "channel" + string(rune('0'+i)),
				getPostsFunc: func(ctx context.Context, limit int) ([]*gateway.TransferPost, error) {
					var posts []*gateway.TransferPost
					for j := 0; j < postsPerGateway; j++ {
						posts = append(posts, createTestTransferPost(j+1, "TOKEN"+string(rune('0'+j)), "100000"))
					}
					return posts, nil
				},
			}
			gateways = append(gateways, mockGW)
		}

		uc := NewTransferUsecase(mockRepo, gateways)
		ctx := context.Background()

		processedCount, errs := uc.ScrapeAndStore(ctx, 10)

		expectedCount := numGateways * postsPerGateway
		if processedCount != expectedCount {
			t.Errorf("processedCount = %d, want %d", processedCount, expectedCount)
		}

		if len(errs) != 0 {
			t.Errorf("errorCount = %d, want 0", len(errs))
		}
	})
}
