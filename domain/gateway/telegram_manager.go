package gateway

import (
	"context"
)

type TelegramClientManager interface {
	Run(ctx context.Context) error
	Stop() error
}