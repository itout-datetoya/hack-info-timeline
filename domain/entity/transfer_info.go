package entity

import "time"

// 送金情報
type TransferInfo struct {
	ID			int64		`db:"id"`
	Token		string		`db:"token"`
	Amount		string		`db:"amount"`
	From		string		`db:"from"`
	To			string		`db:"to"`
	ReportTime	time.Time	`db:"report_time"`
	Tags		[]*Tag	
}