package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/itout-datetoya/hack-info-timeline/domain/entity"

	"github.com/jmoiron/sqlx"
)

// TransferRepository インターフェースを実装する構造体
type transferRepository struct {
	db *sqlx.DB
}

// transferRepository の新しいインスタンスを生成
func NewTransferRepository(db *sqlx.DB) *transferRepository {
	return &transferRepository{db: db}
}

// 指定したタグ名に一致する送金情報を指定の件数取得
func (r *transferRepository) GetInfosByTagNames(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.TransferInfo, error) {
	// 条件に合う送金情報を取得

	// 送金情報テーブルから重複を排除して選択
	query := `
		SELECT DISTINCT
			ti.id, ti.token, ti.amount, ti.from_address, ti.to_address, ti.report_time, ti.message_id
		FROM transfer_infos ti
	`

	args := []interface{}{}

	// タグ名が指定されている場合、JOINとWHERE句を追加
	if len(tagNames) > 0 {
		// 送金情報テーブルと中間テーブルを送金情報IDで結合
		// 中間テーブルとタグテーブルをタグIDで結合
		query += `
			JOIN transfer_info_tags it ON ti.id = it.info_id
			JOIN tags t ON it.tag_id = t.id
			WHERE t.name IN (?)
		`
		// スライスに含まれるタグを持つ送金情報を指定
		var err error
		query, args, err = sqlx.In(query, tagNames)
		if err != nil {
			return nil, fmt.Errorf("failed to expand IN clause: %w", err)
		}
	}

	// タイムスタンプ順に整列、指定件数取得
	query += " ORDER BY ti.report_time DESC LIMIT ?"
	args = append(args, infoNumber)

	// データベースドライバに合わせてプレースホルダーを変換
	query = r.db.Rebind(query)

	// クエリ実行
	var infos []*entity.TransferInfo
	if err := r.db.SelectContext(ctx, &infos, query, args...); err != nil {
		return nil, fmt.Errorf("failed to select infos: %w", err)
	}

	// 送金情報が見つからなければ、処理を終了
	if len(infos) == 0 {
		return infos, nil
	}

	// 取得した送金情報IDに紐づく全てのタグを取得
	infoIDs := make([]int64, len(infos))
	for i, info := range infos {
		infoIDs[i] = info.ID
	}

	// タグテーブルに中間テーブルをタグIDで結合
	tagsQuery := `
		SELECT t.id, t.name, it.info_id
		FROM tags t
		JOIN transfer_info_tags it ON t.id = it.tag_id
		WHERE it.info_id IN (?)
	`

	// タグ取得用の構造体
	type infoTag struct {
		entity.Tag
		InfoID int64 `db:"info_id"`
	}
	var tags []infoTag

	// 取得した送金情報のタグを指定
	query, args, err := sqlx.In(tagsQuery, infoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to expand IN clause for tags: %w", err)
	}

	// データベースドライバに合わせてプレースホルダーを変換
	query = r.db.Rebind(query)

	// クエリ実行
	if err := r.db.SelectContext(ctx, &tags, query, args...); err != nil {
		return nil, fmt.Errorf("failed to select tags for infos: %w", err)
	}

	// 取得したタグを送金情報にマッピング
	tagsByInfoID := make(map[int64][]*entity.Tag)
	for _, t := range tags {
		tag := &entity.Tag{ID: t.ID, Name: t.Name}
		tagsByInfoID[t.InfoID] = append(tagsByInfoID[t.InfoID], tag)
	}

	// 送金情報のスライスにタグをセット
	for _, info := range infos {
		if associatedTags, ok := tagsByInfoID[info.ID]; ok {
			info.Tags = associatedTags
		}
	}

	return infos, nil
}

// 指定したタグ名に一致する送金情報の内、指定した情報より過去から指定の件数取得
func (r *transferRepository) GetPrevInfosByTagNames(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.TransferInfo, error) {
	// 条件に合う送金情報を取得

	// 送金情報テーブルから重複を排除して選択
	query := `
		SELECT DISTINCT
			ti.id, ti.token, ti.amount, ti.from_address, ti.to_address, ti.report_time, ti.message_id
		FROM transfer_infos ti
	`

	args := []interface{}{}

	// タグ名が指定されている場合、JOINとWHERE句を追加
	if len(tagNames) > 0 {
		// 送金情報テーブルと中間テーブルを送金情報IDで結合
		// 中間テーブルとタグテーブルをタグIDで結合
		query += `
			JOIN transfer_info_tags it ON ti.id = it.info_id
			JOIN tags t ON it.tag_id = t.id
			WHERE t.name IN (?)
		`
		// スライスに含まれるタグを持つ送金情報を指定
		var err error
		query, args, err = sqlx.In(query, tagNames)
		if err != nil {
			return nil, fmt.Errorf("failed to expand IN clause: %w", err)
		}
		// すでに取得している情報のIDより過去の情報を取得
		if prevInfoID > 0 {
			query += " AND ti.id < ?"
			args = append(args, prevInfoID)
		}
	} else {
		// すでに取得している情報のIDより過去の情報を取得
		if prevInfoID > 0 {
			query += " WHERE ti.id < ?"
			args = append(args, prevInfoID)
		}
	}

	// タイムスタンプ順に整列、指定件数取得
	query += " ORDER BY ti.report_time DESC LIMIT ?"
	args = append(args, infoNumber)
	// データベースドライバに合わせてプレースホルダーを変換
	query = r.db.Rebind(query)

	// クエリ実行
	var infos []*entity.TransferInfo
	if err := r.db.SelectContext(ctx, &infos, query, args...); err != nil {
		return nil, fmt.Errorf("failed to select infos: %w", err)
	}

	// 送金情報が見つからなければ、処理を終了
	if len(infos) == 0 {
		return infos, nil
	}

	// 取得した送金情報IDに紐づく全てのタグを取得
	infoIDs := make([]int64, len(infos))
	for i, info := range infos {
		infoIDs[i] = info.ID
	}

	// タグテーブルに中間テーブルをタグIDで結合
	tagsQuery := `
		SELECT t.id, t.name, it.info_id
		FROM tags t
		JOIN transfer_info_tags it ON t.id = it.tag_id
		WHERE it.info_id IN (?)
	`

	// タグ取得用の構造体
	type infoTag struct {
		entity.Tag
		InfoID int64 `db:"info_id"`
	}
	var tags []infoTag

	// 取得した送金情報のタグを指定
	query, args, err := sqlx.In(tagsQuery, infoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to expand IN clause for tags: %w", err)
	}

	// データベースドライバに合わせてプレースホルダーを変換
	query = r.db.Rebind(query)

	// クエリ実行
	if err := r.db.SelectContext(ctx, &tags, query, args...); err != nil {
		return nil, fmt.Errorf("failed to select tags for infos: %w", err)
	}

	// 取得したタグを送金情報にマッピング
	tagsByInfoID := make(map[int64][]*entity.Tag)
	for _, t := range tags {
		tag := &entity.Tag{ID: t.ID, Name: t.Name}
		tagsByInfoID[t.InfoID] = append(tagsByInfoID[t.InfoID], tag)
	}

	// 送金情報のスライスにタグをセット
	for _, info := range infos {
		if associatedTags, ok := tagsByInfoID[info.ID]; ok {
			info.Tags = associatedTags
		}
	}

	return infos, nil
}

// 存在するすべてのタグを取得
func (r *transferRepository) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	var tags []*entity.Tag
	query := "SELECT id, name FROM tags ORDER BY name"
	if err := r.db.SelectContext(ctx, &tags, query); err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, nil
}

// 新しい送金情報と関連タグをトランザクション内で保存
func (r *transferRepository) StoreInfo(ctx context.Context, info *entity.TransferInfo, tagNames []string) (int64, error) {
	// トランザクションを開始
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	// 関数を抜ける際にエラーがあればロールバック
	defer tx.Rollback()

	// 送金情報を保存するクエリ文を設定
	stmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO transfer_infos (token, amount, from_address, to_address, report_time, message_id)
		VALUES (:token, :amount, :from_address, :to_address, :report_time, :message_id)
		RETURNING id
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare info statement: %w", err)
	}
	defer stmt.Close()

	var infoID int64
	// 送金情報を `transfer_infos` テーブルに保存
	// 送金情報のIDを取得
	if err := stmt.GetContext(ctx, &infoID, info); err != nil {
		return 0, fmt.Errorf("failed to execute info statement: %w", err)
	}

	// タグを `tags` テーブルに保存
	tagIDs := []int64{}
	for _, name := range tagNames {
		var tagID int64
		// タグが既に存在するか確認
		err := tx.GetContext(ctx, &tagID, "SELECT id FROM tags WHERE name = $1", name)
		if err != nil {
			// 存在しない場合、新しく保存してIDを取得
			err = tx.QueryRowxContext(ctx, "INSERT INTO tags (name) VALUES ($1) RETURNING id", name).Scan(&tagID)
			if err != nil {
				return 0, fmt.Errorf("failed to insert tag: %w", err)
			}
		}
		tagIDs = append(tagIDs, tagID)
	}

	// 中間テーブル `transfer_info_tags` に送金情報とタグの関連を保存
	for _, tagID := range tagIDs {
		_, err := tx.ExecContext(ctx, "INSERT INTO transfer_info_tags (info_id, tag_id) VALUES ($1, $2)", infoID, tagID)
		if err != nil {
			return 0, fmt.Errorf("failed to insert into transfer_info_tags: %w", err)
		}
	}

	// トランザクションをコミットして変更を確定
	return infoID, tx.Commit()
}

// チャンネル情報をトランザクション内で保存
func (r *transferRepository) StoreChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error {
	// トランザクションを開始
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// 関数を抜ける際にエラーがあればロールバック
	defer tx.Rollback()

	// チャンネル情報を保存するクエリ文を設定
	stmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO telegram_channel (username, last_message_id)
		VALUES (:username, :last_message_id)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// チャンネル情報を `telegram_channel` テーブルに保存
	if _, err := stmt.ExecContext(ctx, channelStatus); err != nil {
		return fmt.Errorf("failed to execute statement: %w", err)
	}

	// トランザクションをコミットして変更を確定
	return tx.Commit()
}

// チャンネル情報をトランザクション内で更新
func (r *transferRepository) UpdateChannelStatus(ctx context.Context, channelStatus *entity.TelegramChannel) error {
	// トランザクションを開始
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// 関数を抜ける際にエラーがあればロールバック
	defer tx.Rollback()

	// チャンネル情報を保存するクエリ文を設定
	stmt, err := tx.PrepareNamedContext(ctx, `
		UPDATE telegram_channel
		SET last_message_id = :last_message_id
		WHERE username = :username
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// チャンネル情報を `telegram_channel` テーブルに保存
	if _, err := stmt.ExecContext(ctx, channelStatus); err != nil {
		return fmt.Errorf("failed to execute statement: %w", err)
	}

	// トランザクションをコミットして変更を確定
	return tx.Commit()
}

// usernameで指定されたチャンネル情報を1件取得
func (r *transferRepository) GetChannelStatusByUsername(ctx context.Context, username string) (*entity.TelegramChannel, error) {
	// 取得した結果を格納するための変数を宣言
	var channel *entity.TelegramChannel

	// チャンネル情報を取得するクエリ文
	query := `
		SELECT username, last_message_id
		FROM telegram_channel
		WHERE username = ?
	`

	query = r.db.Rebind(query)
	
	// sqlx.GetContext を使用して、結果を channel 変数に直接マッピングします。
	// クエリのプレースホルダは、使用するDBドライバに合わせて '?' や '$1' などを選択してください。
	if err := r.db.GetContext(ctx, &channel, query, username); err != nil {
		// 該当するレコードが1件もなかった場合、sql.ErrNoRows が返される
		if errors.Is(err, sql.ErrNoRows) {
			// 見つからなかった場合は、nil と nil を返す
			return nil, nil
		}
		// その他のデータベースエラー
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	return channel, nil
}
