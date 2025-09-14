package entity

// 各種情報に付けられる検索タグ
type TelegramChannel struct {
	ChannelUsername string `db:"username"`
	LastMessageID   int    `db:"last_message_id"`
}
