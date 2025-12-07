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

// mockHackingRepository は HackingRepository インターフェースのモック実装
type mockHackingRepository struct {
	getInfosByTagNamesFunc         func(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.HackingInfo, error)
	getPrevInfosByTagNamesFunc     func(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.HackingInfo, error)
	getAllTagsFunc                 func(ctx context.Context) ([]*entity.Tag, error)
	setTagToCacheFunc              func(ctx context.Context) error
	storeInfoFunc                  func(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error)
	storeChannelStatusFunc         func(ctx context.Context, channelStatus *entity.TelegramChannel) error
	updateChannelStatusFunc        func(ctx context.Context, channelStatus *entity.TelegramChannel) error
	getChannelStatusByUsernameFunc func(ctx context.Context, username string) (*entity.TelegramChannel, error)
}

func (m *mockHackingRepository) GetInfosByTagNames(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.HackingInfo, error) {
	if m.getInfosByTagNamesFunc != nil {
		return m.getInfosByTagNamesFunc(ctx, tagNames, infoNumber)
	}
	return nil, nil
}

func (m *mockHackingRepository) GetPrevInfosByTagNames(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.HackingInfo, error) {
	if m.getPrevInfosByTagNamesFunc != nil {
		return m.getPrevInfosByTagNamesFunc(ctx, tagNames, prevInfoID, infoNumber)
	}
	return nil, nil
}

func (m *mockHackingRepository) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	if m.getAllTagsFunc != nil {
		return m.getAllTagsFunc(ctx)
	}
	return nil, nil
}

func (m *mockHackingRepository) SetTagToCache(ctx context.Context) error {
	if m.setTagToCacheFunc != nil {
		return m.setTagToCacheFunc(ctx)
	}
	return nil
}

func (m *mockHackingRepository) StoreInfo(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error) {
	if m.storeInfoFunc != nil {
		return m.storeInfoFunc(ctx, info, tagNames)
	}
	return 0, nil
}

func (m *mockHackingRepository) StoreChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error {
	if m.storeChannelStatusFunc != nil {
		return m.storeChannelStatusFunc(ctx, channelStatus)
	}
	return nil
}

func (m *mockHackingRepository) UpdateChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error {
	if m.updateChannelStatusFunc != nil {
		return m.updateChannelStatusFunc(ctx, channelStatus)
	}
	return nil
}

func (m *mockHackingRepository) GetChannelStatusByUsername(ctx context.Context, username string) (*entity.TelegramChannel, error) {
	if m.getChannelStatusByUsernameFunc != nil {
		return m.getChannelStatusByUsernameFunc(ctx, username)
	}
	return nil, nil
}

// mockTelegramHackingPostGateway は TelegramHackingPostGateway インターフェースのモック実装
type mockTelegramHackingPostGateway struct {
	channelUsername     string
	lastMessageID       int
	getPostsFunc        func(ctx context.Context, limit int) ([]*gateway.HackingPost, error)
	getPostsOver100Func func(ctx context.Context, limit int) ([]*gateway.HackingPost, error)
	mu                  sync.Mutex
}

func (m *mockTelegramHackingPostGateway) SetLastMessageID(lastMessageID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastMessageID = lastMessageID
}

func (m *mockTelegramHackingPostGateway) LastMessageID() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastMessageID
}

func (m *mockTelegramHackingPostGateway) ChannelUsername() string {
	return m.channelUsername
}

func (m *mockTelegramHackingPostGateway) GetPosts(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
	if m.getPostsFunc != nil {
		return m.getPostsFunc(ctx, limit)
	}
	return nil, nil
}

func (m *mockTelegramHackingPostGateway) GetPostsOver100(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
	if m.getPostsOver100Func != nil {
		return m.getPostsOver100Func(ctx, limit)
	}
	return nil, nil
}

// mockGeminiGateway は GeminiGateway インターフェースのモック実装
type mockGeminiGateway struct {
	analyzeAndExtractFunc func(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error)
	stopFunc              func() error
}

func (m *mockGeminiGateway) AnalyzeAndExtract(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error) {
	if m.analyzeAndExtractFunc != nil {
		return m.analyzeAndExtractFunc(ctx, post)
	}
	return nil, nil
}

func (m *mockGeminiGateway) Stop() error {
	if m.stopFunc != nil {
		return m.stopFunc()
	}
	return nil
}

// ==================== Test Helper Functions ====================

func createTestHackingPost(messageID int, txHash string) *gateway.HackingPost {
	return &gateway.HackingPost{
		Text:       "Test post",
		Network:    "Ethereum",
		Amount:     "$1000000",
		TxHash:     txHash,
		ReportTime: time.Now(),
		MessageID:  messageID,
	}
}

func createTestHackingInfo(id int64, txHash string) *entity.HackingInfo {
	return &entity.HackingInfo{
		ID:         id,
		Protocol:   "TestProtocol",
		Network:    "Ethereum",
		Amount:     "$1000000",
		TxHash:     txHash,
		ReportTime: time.Now(),
		MessageID:  100,
		Tags:       []*entity.Tag{{ID: 1, Name: "DeFi"}},
	}
}

// ==================== Simple Method Tests ====================

func TestGetLatestTimeline(t *testing.T) {
	tests := []struct {
		name       string
		tagNames   []string
		infoNumber int
		mockResult []*entity.HackingInfo
		mockError  error
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "success case",
			tagNames:   []string{"DeFi", "Hack"},
			infoNumber: 10,
			mockResult: []*entity.HackingInfo{
				createTestHackingInfo(1, "0xabc123"),
				createTestHackingInfo(2, "0xdef456"),
			},
			mockError: nil,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:       "empty result",
			tagNames:   []string{"NonExistent"},
			infoNumber: 10,
			mockResult: []*entity.HackingInfo{},
			mockError:  nil,
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "repository error",
			tagNames:   []string{"DeFi"},
			infoNumber: 10,
			mockResult: nil,
			mockError:  errors.New("database error"),
			wantErr:    true,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockHackingRepository{
				getInfosByTagNamesFunc: func(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.HackingInfo, error) {
					return tt.mockResult, tt.mockError
				},
			}

			uc := NewHackingUsecase(mockRepo, nil, nil)
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

func TestGetPrevTimeline(t *testing.T) {
	tests := []struct {
		name       string
		tagNames   []string
		prevInfoID int64
		infoNumber int
		mockResult []*entity.HackingInfo
		mockError  error
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "success case",
			tagNames:   []string{"DeFi"},
			prevInfoID: 100,
			infoNumber: 5,
			mockResult: []*entity.HackingInfo{
				createTestHackingInfo(99, "0xabc123"),
				createTestHackingInfo(98, "0xdef456"),
			},
			mockError: nil,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:       "no previous data",
			tagNames:   []string{"DeFi"},
			prevInfoID: 1,
			infoNumber: 5,
			mockResult: []*entity.HackingInfo{},
			mockError:  nil,
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "repository error",
			tagNames:   []string{"DeFi"},
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
			mockRepo := &mockHackingRepository{
				getPrevInfosByTagNamesFunc: func(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.HackingInfo, error) {
					return tt.mockResult, tt.mockError
				},
			}

			uc := NewHackingUsecase(mockRepo, nil, nil)
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

func TestGetAllTags(t *testing.T) {
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
				{ID: 1, Name: "DeFi"},
				{ID: 2, Name: "Hack"},
				{ID: 3, Name: "Bridge"},
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
			mockRepo := &mockHackingRepository{
				getAllTagsFunc: func(ctx context.Context) ([]*entity.Tag, error) {
					return tt.mockResult, tt.mockError
				},
			}

			uc := NewHackingUsecase(mockRepo, nil, nil)
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

func TestSetTagToCache(t *testing.T) {
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
			mockRepo := &mockHackingRepository{
				setTagToCacheFunc: func(ctx context.Context) error {
					return tt.mockError
				},
			}

			uc := NewHackingUsecase(mockRepo, nil, nil)
			ctx := context.Background()

			err := uc.SetTagToCache(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetTagToCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ==================== State Management Tests ====================

func TestSetLastMessageIDToGateway(t *testing.T) {
	tests := []struct {
		name               string
		channelUsernames   []string
		existingStatus     map[string]*entity.TelegramChannel
		getStatusError     error
		storeStatusError   error
		wantErr            bool
		expectedLastMsgIDs map[string]int
	}{
		{
			name:             "existing channels",
			channelUsernames: []string{"channel1", "channel2"},
			existingStatus: map[string]*entity.TelegramChannel{
				"channel1": {ChannelUsername: "channel1", LastMessageID: 100},
				"channel2": {ChannelUsername: "channel2", LastMessageID: 200},
			},
			getStatusError:   nil,
			storeStatusError: nil,
			wantErr:          false,
			expectedLastMsgIDs: map[string]int{
				"channel1": 100,
				"channel2": 200,
			},
		},
		{
			name:             "new channels",
			channelUsernames: []string{"newchannel"},
			existingStatus:   map[string]*entity.TelegramChannel{},
			getStatusError:   nil,
			storeStatusError: nil,
			wantErr:          false,
			expectedLastMsgIDs: map[string]int{
				"newchannel": 0,
			},
		},
		{
			name:               "get status error",
			channelUsernames:   []string{"channel1"},
			existingStatus:     nil,
			getStatusError:     errors.New("database error"),
			storeStatusError:   nil,
			wantErr:            true,
			expectedLastMsgIDs: nil,
		},
		{
			name:               "store status error for new channel",
			channelUsernames:   []string{"newchannel"},
			existingStatus:     map[string]*entity.TelegramChannel{},
			getStatusError:     nil,
			storeStatusError:   errors.New("store error"),
			wantErr:            true,
			expectedLastMsgIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockHackingRepository{
				getChannelStatusByUsernameFunc: func(ctx context.Context, username string) (*entity.TelegramChannel, error) {
					if tt.getStatusError != nil {
						return nil, tt.getStatusError
					}
					return tt.existingStatus[username], nil
				},
				storeChannelStatusFunc: func(ctx context.Context, channelStatus *entity.TelegramChannel) error {
					return tt.storeStatusError
				},
			}

			var gateways []gateway.TelegramHackingPostGateway
			mockGateways := make(map[string]*mockTelegramHackingPostGateway)
			for _, username := range tt.channelUsernames {
				mockGW := &mockTelegramHackingPostGateway{
					channelUsername: username,
				}
				mockGateways[username] = mockGW
				gateways = append(gateways, mockGW)
			}

			uc := NewHackingUsecase(mockRepo, gateways, nil)
			ctx := context.Background()

			err := uc.SetLastMessageIDToGateway(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetLastMessageIDToGateway() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.expectedLastMsgIDs != nil {
				for username, expectedID := range tt.expectedLastMsgIDs {
					if mockGW, ok := mockGateways[username]; ok {
						actualID := mockGW.LastMessageID()
						if actualID != expectedID {
							t.Errorf("Gateway %s LastMessageID = %d, want %d", username, actualID, expectedID)
						}
					}
				}
			}
		})
	}
}

func TestStoreLastMessageID(t *testing.T) {
	tests := []struct {
		name              string
		channelUsernames  []string
		currentLastMsgIDs map[string]int
		existingStatus    map[string]*entity.TelegramChannel
		getStatusError    error
		storeStatusError  error
		updateStatusError error
		wantErr           bool
	}{
		{
			name:             "update existing channels",
			channelUsernames: []string{"channel1", "channel2"},
			currentLastMsgIDs: map[string]int{
				"channel1": 150,
				"channel2": 250,
			},
			existingStatus: map[string]*entity.TelegramChannel{
				"channel1": {ChannelUsername: "channel1", LastMessageID: 100},
				"channel2": {ChannelUsername: "channel2", LastMessageID: 200},
			},
			getStatusError:    nil,
			storeStatusError:  nil,
			updateStatusError: nil,
			wantErr:           false,
		},
		{
			name:             "store new channels",
			channelUsernames: []string{"newchannel"},
			currentLastMsgIDs: map[string]int{
				"newchannel": 50,
			},
			existingStatus:    map[string]*entity.TelegramChannel{},
			getStatusError:    nil,
			storeStatusError:  nil,
			updateStatusError: nil,
			wantErr:           false,
		},
		{
			name:             "get status error",
			channelUsernames: []string{"channel1"},
			currentLastMsgIDs: map[string]int{
				"channel1": 150,
			},
			existingStatus:    nil,
			getStatusError:    errors.New("database error"),
			storeStatusError:  nil,
			updateStatusError: nil,
			wantErr:           true,
		},
		{
			name:             "update status error",
			channelUsernames: []string{"channel1"},
			currentLastMsgIDs: map[string]int{
				"channel1": 150,
			},
			existingStatus: map[string]*entity.TelegramChannel{
				"channel1": {ChannelUsername: "channel1", LastMessageID: 100},
			},
			getStatusError:    nil,
			storeStatusError:  nil,
			updateStatusError: errors.New("update error"),
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockHackingRepository{
				getChannelStatusByUsernameFunc: func(ctx context.Context, username string) (*entity.TelegramChannel, error) {
					if tt.getStatusError != nil {
						return nil, tt.getStatusError
					}
					return tt.existingStatus[username], nil
				},
				storeChannelStatusFunc: func(ctx context.Context, channelStatus *entity.TelegramChannel) error {
					return tt.storeStatusError
				},
				updateChannelStatusFunc: func(ctx context.Context, channelStatus *entity.TelegramChannel) error {
					return tt.updateStatusError
				},
			}

			var gateways []gateway.TelegramHackingPostGateway
			for _, username := range tt.channelUsernames {
				mockGW := &mockTelegramHackingPostGateway{
					channelUsername: username,
					lastMessageID:   tt.currentLastMsgIDs[username],
				}
				gateways = append(gateways, mockGW)
			}

			uc := NewHackingUsecase(mockRepo, gateways, nil)
			ctx := context.Background()

			err := uc.StoreLastMessageID(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("StoreLastMessageID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ==================== Process Single Post Tests ====================

func TestProcessSinglePost(t *testing.T) {
	tests := []struct {
		name            string
		post            *gateway.HackingPost
		extractedInfo   *gateway.ExtractedHackingInfo
		geminiError     error
		storeError      error
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "success case",
			post: createTestHackingPost(100, "0xabc123"),
			extractedInfo: &gateway.ExtractedHackingInfo{
				Protocol: "Uniswap",
				Network:  "Ethereum",
				Amount:   "$1000000",
				TxHash:   "0xabc123",
				TagNames: []string{"DeFi", "DEX"},
			},
			geminiError:     nil,
			storeError:      nil,
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name:            "gemini analysis error",
			post:            createTestHackingPost(100, "0xabc123"),
			extractedInfo:   nil,
			geminiError:     errors.New("API rate limit"),
			storeError:      nil,
			wantErr:         true,
			wantErrContains: "gemini analysis failed",
		},
		{
			name: "database store error",
			post: createTestHackingPost(100, "0xabc123"),
			extractedInfo: &gateway.ExtractedHackingInfo{
				Protocol: "Uniswap",
				Network:  "Ethereum",
				Amount:   "$1000000",
				TxHash:   "0xabc123",
				TagNames: []string{"DeFi"},
			},
			geminiError:     nil,
			storeError:      errors.New("constraint violation"),
			wantErr:         true,
			wantErrContains: "database store failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockHackingRepository{
				storeInfoFunc: func(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error) {
					if tt.storeError != nil {
						return 0, tt.storeError
					}
					return 1, nil
				},
			}

			mockGemini := &mockGeminiGateway{
				analyzeAndExtractFunc: func(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error) {
					if tt.geminiError != nil {
						return nil, tt.geminiError
					}
					return tt.extractedInfo, nil
				},
			}

			uc := NewHackingUsecase(mockRepo, nil, mockGemini)
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

func TestScrapeAndStore(t *testing.T) {
	tests := []struct {
		name               string
		limit              int
		numGateways        int
		postsPerGateway    [][]*gateway.HackingPost
		getPostsErrors     []error
		processErrors      map[string]error // txHash -> error
		wantProcessedCount int
		wantErrorCount     int
		wantLastMessageIDs []int
	}{
		{
			name:        "success case with single gateway",
			limit:       10,
			numGateways: 1,
			postsPerGateway: [][]*gateway.HackingPost{
				{
					createTestHackingPost(101, "0xabc123"),
					createTestHackingPost(102, "0xdef456"),
					createTestHackingPost(103, "0xghi789"),
				},
			},
			getPostsErrors:     []error{nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 3,
			wantErrorCount:     0,
			wantLastMessageIDs: []int{103},
		},
		{
			name:        "success with multiple gateways",
			limit:       10,
			numGateways: 2,
			postsPerGateway: [][]*gateway.HackingPost{
				{
					createTestHackingPost(101, "0xaaa111"),
					createTestHackingPost(102, "0xbbb222"),
				},
				{
					createTestHackingPost(201, "0xccc333"),
					createTestHackingPost(202, "0xddd444"),
					createTestHackingPost(203, "0xeee555"),
				},
			},
			getPostsErrors:     []error{nil, nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 5,
			wantErrorCount:     0,
			wantLastMessageIDs: []int{102, 203},
		},
		{
			name:        "partial processing errors",
			limit:       10,
			numGateways: 1,
			postsPerGateway: [][]*gateway.HackingPost{
				{
					createTestHackingPost(101, "0xabc123"),
					createTestHackingPost(102, "0xdef456"),
					createTestHackingPost(103, "0xghi789"),
				},
			},
			getPostsErrors: []error{nil},
			processErrors: map[string]error{
				"0xdef456": errors.New("gemini error"),
			},
			wantProcessedCount: 2,
			wantErrorCount:     1,
			wantLastMessageIDs: []int{103},
		},
		{
			name:        "get posts error",
			limit:       10,
			numGateways: 2,
			postsPerGateway: [][]*gateway.HackingPost{
				nil,
				nil,
			},
			getPostsErrors:     []error{errors.New("telegram error"), nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 0,
			wantErrorCount:     1,
			wantLastMessageIDs: []int{0, 0},
		},
		{
			name:               "no posts to process",
			limit:              10,
			numGateways:        1,
			postsPerGateway:    [][]*gateway.HackingPost{{}},
			getPostsErrors:     []error{nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 0,
			wantErrorCount:     0,
			wantLastMessageIDs: []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockHackingRepository{
				storeInfoFunc: func(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error) {
					if err, exists := tt.processErrors[info.TxHash]; exists {
						return 0, err
					}
					return 1, nil
				},
			}

			mockGemini := &mockGeminiGateway{
				analyzeAndExtractFunc: func(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error) {
					if err, exists := tt.processErrors[post.TxHash]; exists {
						return nil, err
					}
					return &gateway.ExtractedHackingInfo{
						Protocol: "TestProtocol",
						Network:  "Ethereum",
						Amount:   "$1000000",
						TxHash:   post.TxHash,
						TagNames: []string{"DeFi"},
					}, nil
				},
			}

			var gateways []gateway.TelegramHackingPostGateway
			for i := 0; i < tt.numGateways; i++ {
				idx := i
				mockGW := &mockTelegramHackingPostGateway{
					channelUsername: "channel" + string(rune('1'+i)),
					getPostsFunc: func(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
						if tt.getPostsErrors[idx] != nil {
							return nil, tt.getPostsErrors[idx]
						}
						return tt.postsPerGateway[idx], nil
					},
				}
				gateways = append(gateways, mockGW)
			}

			uc := NewHackingUsecase(mockRepo, gateways, mockGemini)
			ctx := context.Background()

			processedCount, errs := uc.ScrapeAndStore(ctx, tt.limit)

			if processedCount != tt.wantProcessedCount {
				t.Errorf("ScrapeAndStore() processedCount = %d, want %d", processedCount, tt.wantProcessedCount)
			}

			if len(errs) != tt.wantErrorCount {
				t.Errorf("ScrapeAndStore() errorCount = %d, want %d", len(errs), tt.wantErrorCount)
			}

			// Verify LastMessageID updates
			for i, gw := range gateways {
				actualID := gw.LastMessageID()
				expectedID := tt.wantLastMessageIDs[i]
				if actualID != expectedID {
					t.Errorf("Gateway %d LastMessageID = %d, want %d", i, actualID, expectedID)
				}
			}
		})
	}
}

func TestScrapeAndStore_RetryQueue(t *testing.T) {
	t.Run("retry queue processing", func(t *testing.T) {
		mockRepo := &mockHackingRepository{
			storeInfoFunc: func(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error) {
				return 1, nil
			},
		}

		callCount := 0
		mockGemini := &mockGeminiGateway{
			analyzeAndExtractFunc: func(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error) {
				callCount++
				// First call fails, subsequent calls succeed
				if callCount == 1 {
					return nil, errors.New("temporary error")
				}
				return &gateway.ExtractedHackingInfo{
					Protocol: "TestProtocol",
					Network:  "Ethereum",
					Amount:   "$1000000",
					TxHash:   post.TxHash,
					TagNames: []string{"DeFi"},
				}, nil
			},
		}

		mockGW := &mockTelegramHackingPostGateway{
			channelUsername: "channel1",
			getPostsFunc: func(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
				return []*gateway.HackingPost{
					createTestHackingPost(101, "0xabc123"),
				}, nil
			},
		}

		uc := NewHackingUsecase(mockRepo, []gateway.TelegramHackingPostGateway{mockGW}, mockGemini)
		ctx := context.Background()

		// First run: should fail and add to retry queue
		processedCount1, errs1 := uc.ScrapeAndStore(ctx, 10)
		if processedCount1 != 0 {
			t.Errorf("First run: processedCount = %d, want 0", processedCount1)
		}
		if len(errs1) != 1 {
			t.Errorf("First run: errorCount = %d, want 1", len(errs1))
		}

		// Verify retry queue has 1 item
		if len(uc.retryQueue[0]) != 1 {
			t.Errorf("Retry queue length = %d, want 1", len(uc.retryQueue[0]))
		}

		// Second run: should process retry queue successfully
		mockGW.getPostsFunc = func(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
			return []*gateway.HackingPost{}, nil // No new posts
		}

		processedCount2, errs2 := uc.ScrapeAndStore(ctx, 10)
		if processedCount2 != 1 {
			t.Errorf("Second run: processedCount = %d, want 1", processedCount2)
		}
		if len(errs2) != 0 {
			t.Errorf("Second run: errorCount = %d, want 0", len(errs2))
		}

		// Verify retry queue is now empty
		if len(uc.retryQueue[0]) != 0 {
			t.Errorf("Retry queue length after success = %d, want 0", len(uc.retryQueue[0]))
		}
	})
}

func TestScrapeAndStore_MessageIDUpdate(t *testing.T) {
	t.Run("message ID updates to maximum", func(t *testing.T) {
		mockRepo := &mockHackingRepository{
			storeInfoFunc: func(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error) {
				return 1, nil
			},
		}

		mockGemini := &mockGeminiGateway{
			analyzeAndExtractFunc: func(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error) {
				return &gateway.ExtractedHackingInfo{
					Protocol: "TestProtocol",
					Network:  "Ethereum",
					Amount:   "$1000000",
					TxHash:   post.TxHash,
					TagNames: []string{"DeFi"},
				}, nil
			},
		}

		mockGW := &mockTelegramHackingPostGateway{
			channelUsername: "channel1",
			lastMessageID:   50, // Starting ID
			getPostsFunc: func(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
				return []*gateway.HackingPost{
					createTestHackingPost(55, "0xaaa111"),
					createTestHackingPost(101, "0xbbb222"), // Maximum
					createTestHackingPost(75, "0xccc333"),
					createTestHackingPost(60, "0xddd444"),
				}, nil
			},
		}

		uc := NewHackingUsecase(mockRepo, []gateway.TelegramHackingPostGateway{mockGW}, mockGemini)
		ctx := context.Background()

		_, _ = uc.ScrapeAndStore(ctx, 10)

		// Should update to maximum message ID
		actualID := mockGW.LastMessageID()
		expectedID := 101
		if actualID != expectedID {
			t.Errorf("LastMessageID = %d, want %d", actualID, expectedID)
		}
	})
}

// ==================== InitialScrapeAndStore Tests ====================

func TestInitialScrapeAndStore(t *testing.T) {
	tests := []struct {
		name               string
		limit              int
		numGateways        int
		postsPerGateway    [][]*gateway.HackingPost
		getPostsErrors     []error
		processErrors      map[string]error
		wantProcessedCount int
		wantErrorCount     int
	}{
		{
			name:        "success with over 100 posts",
			limit:       150,
			numGateways: 1,
			postsPerGateway: [][]*gateway.HackingPost{
				{
					createTestHackingPost(101, "0xaaa111"),
					createTestHackingPost(102, "0xbbb222"),
					createTestHackingPost(103, "0xccc333"),
				},
			},
			getPostsErrors:     []error{nil},
			processErrors:      map[string]error{},
			wantProcessedCount: 3,
			wantErrorCount:     0,
		},
		{
			name:        "partial errors with large dataset",
			limit:       200,
			numGateways: 1,
			postsPerGateway: [][]*gateway.HackingPost{
				{
					createTestHackingPost(101, "0xaaa111"),
					createTestHackingPost(102, "0xbbb222"),
					createTestHackingPost(103, "0xccc333"),
					createTestHackingPost(104, "0xddd444"),
					createTestHackingPost(105, "0xeee555"),
				},
			},
			getPostsErrors: []error{nil},
			processErrors: map[string]error{
				"0xbbb222": errors.New("gemini error"),
				"0xddd444": errors.New("store error"),
			},
			wantProcessedCount: 3,
			wantErrorCount:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockHackingRepository{
				storeInfoFunc: func(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error) {
					if err, exists := tt.processErrors[info.TxHash]; exists {
						return 0, err
					}
					return 1, nil
				},
			}

			mockGemini := &mockGeminiGateway{
				analyzeAndExtractFunc: func(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error) {
					if err, exists := tt.processErrors[post.TxHash]; exists {
						return nil, err
					}
					return &gateway.ExtractedHackingInfo{
						Protocol: "TestProtocol",
						Network:  "Ethereum",
						Amount:   "$1000000",
						TxHash:   post.TxHash,
						TagNames: []string{"DeFi"},
					}, nil
				},
			}

			var gateways []gateway.TelegramHackingPostGateway
			for i := 0; i < tt.numGateways; i++ {
				idx := i
				mockGW := &mockTelegramHackingPostGateway{
					channelUsername: "channel" + string(rune('1'+i)),
					getPostsOver100Func: func(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
						if tt.getPostsErrors[idx] != nil {
							return nil, tt.getPostsErrors[idx]
						}
						return tt.postsPerGateway[idx], nil
					},
				}
				gateways = append(gateways, mockGW)
			}

			uc := NewHackingUsecase(mockRepo, gateways, mockGemini)
			ctx := context.Background()

			processedCount, errs := uc.InitialScrapeAndStore(ctx, tt.limit)

			if processedCount != tt.wantProcessedCount {
				t.Errorf("InitialScrapeAndStore() processedCount = %d, want %d", processedCount, tt.wantProcessedCount)
			}

			if len(errs) != tt.wantErrorCount {
				t.Errorf("InitialScrapeAndStore() errorCount = %d, want %d", len(errs), tt.wantErrorCount)
			}
		})
	}
}

// ==================== Concurrency Tests ====================

func TestScrapeAndStore_Concurrency(t *testing.T) {
	t.Run("concurrent gateway processing", func(t *testing.T) {
		numGateways := 10
		postsPerGateway := 5

		mockRepo := &mockHackingRepository{
			storeInfoFunc: func(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error) {
				// Simulate some processing time
				time.Sleep(1 * time.Millisecond)
				return 1, nil
			},
		}

		mockGemini := &mockGeminiGateway{
			analyzeAndExtractFunc: func(ctx context.Context, post *gateway.HackingPost) (*gateway.ExtractedHackingInfo, error) {
				return &gateway.ExtractedHackingInfo{
					Protocol: "TestProtocol",
					Network:  "Ethereum",
					Amount:   "$1000000",
					TxHash:   post.TxHash,
					TagNames: []string{"DeFi"},
				}, nil
			},
		}

		var gateways []gateway.TelegramHackingPostGateway
		for i := 0; i < numGateways; i++ {
			mockGW := &mockTelegramHackingPostGateway{
				channelUsername: "channel" + string(rune('0'+i)),
				getPostsFunc: func(ctx context.Context, limit int) ([]*gateway.HackingPost, error) {
					var posts []*gateway.HackingPost
					for j := 0; j < postsPerGateway; j++ {
						posts = append(posts, createTestHackingPost(j+1, "0xtx"+string(rune('0'+j))))
					}
					return posts, nil
				},
			}
			gateways = append(gateways, mockGW)
		}

		uc := NewHackingUsecase(mockRepo, gateways, mockGemini)
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

// ==================== Helper Functions ====================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
