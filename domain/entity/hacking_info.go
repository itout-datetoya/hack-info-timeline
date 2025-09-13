package entity

import "time"

// ハッキング情報
type HackingInfo struct {
	ID         int64     `db:"id"`
	Protocol   string    `db:"protocol"`
	Network    string    `db:"network"`
	Amount     string    `db:"amount"`
	TxHash     string    `db:"tx_hash"`
	ReportTime time.Time `db:"report_time"`
	MessageID  int       `db:"message_id"`
	Tags       []*Tag
}
