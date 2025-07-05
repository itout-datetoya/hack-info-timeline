package entity

// 各種情報に付けられる検索タグ
type Tag struct {
	ID		int64	`db:"id"`
	Name	string	`db:"name"`
}