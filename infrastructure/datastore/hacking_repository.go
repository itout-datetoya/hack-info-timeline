package datastore

import (
	"context"
	"fmt"
	"github.com/itout-datetoya/hack-info-timeline/domain/entity"

	"github.com/jmoiron/sqlx"
)

// HackingRepository インターフェースを実装する構造体
type hackingRepository struct {
	db *sqlx.DB
}

// hackingRepository の新しいインスタンスを生成
func NewHackingRepository(db *sqlx.DB) *hackingRepository {
	return &hackingRepository{db: db}
}

// 指定したタグ名に一致する情報を指定の件数取得
func (r *hackingRepository) GetInfosByTagNames(ctx context.Context, tagNames []string, infoNumber int) ([]*entity.HackingInfo, error) {
	// 条件に合うハッキング情報を取得
	
	// ハッキング情報テーブルから重複を排除して選択
	query := `
		SELECT DISTINCT
			hi.id, hi.protocol, hi.network, hi.amount, hi.tx_hash, hi.report_time
		FROM hacking_infos hi
	`

	args := []interface{}{}

	// タグ名が指定されている場合、JOINとWHERE句を追加
	if len(tagNames) > 0 {
		// ハッキング情報テーブルと中間テーブルをハッキング情報IDで結合
		// 中間テーブルとタグテーブルをタグIDで結合
		query += `
			JOIN hacking_info_tags it ON hi.id = it.info_id
			JOIN tags t ON it.tag_id = t.id
			WHERE t.name IN (?)
		`
		// スライスに含まれるタグを持つハッキング情報を指定
		var err error
		query, args, err = sqlx.In(query, tagNames)
		if err != nil {
			return nil, fmt.Errorf("failed to expand IN clause: %w", err)
		}
	}


	// ID順に整列、指定件数取得
	query += " ORDER BY hi.id DESC LIMIT ?"
	args = append(args, infoNumber)

	// データベースドライバに合わせてプレースホルダーを変換
	query = r.db.Rebind(query)

	// クエリ実行
	var infos []*entity.HackingInfo
	if err := r.db.SelectContext(ctx, &infos, query, args...); err != nil {
		return nil, fmt.Errorf("failed to select infos: %w", err)
	}

	// ハッキング情報が見つからなければ、処理を終了
	if len(infos) == 0 {
		return infos, nil
	}

	// 取得したハッキング情報IDに紐づく全てのタグを取得
	infoIDs := make([]int64, len(infos))
	for i, info := range infos {
		infoIDs[i] = info.ID
	}

	// タグテーブルに中間テーブルをタグIDで結合
	tagsQuery := `
		SELECT t.id, t.name, it.info_id
		FROM tags t
		JOIN hacking_info_tags it ON t.id = it.tag_id
		WHERE it.info_id IN (?)
	`

	// タグ取得用の構造体
	type infoTag struct {
		entity.Tag
		InfoID int64 `db:"info_id"`
	}
	var tags []infoTag

	// 取得したハッキング情報のタグを指定
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

	// 取得したタグをハッキング情報にマッピング
	tagsByInfoID := make(map[int64][]*entity.Tag)
	for _, t := range tags {
		tag := &entity.Tag{ID: t.ID, Name: t.Name}
		tagsByInfoID[t.InfoID] = append(tagsByInfoID[t.InfoID], tag)
	}

	// ハッキング情報のスライスにタグをセット
	for _, info := range infos {
		if associatedTags, ok := tagsByInfoID[info.ID]; ok {
			info.Tags = associatedTags
		}
	}

	return infos, nil
}

// 指定したタグ名に一致する情報の内、指定した情報より過去から指定の件数取得
func (r *hackingRepository) GetPrevInfosByTagNames(ctx context.Context, tagNames []string, prevInfoID int64, infoNumber int) ([]*entity.HackingInfo, error) {
	// 条件に合うハッキング情報を取得
	
	// ハッキング情報テーブルから重複を排除して選択
	query := `
		SELECT DISTINCT
			hi.id, hi.protocol, hi.network, hi.amount, hi.tx_hash, hi.report_time
		FROM hacking_infos hi
	`

	args := []interface{}{}

	// タグ名が指定されている場合、JOINとWHERE句を追加
	if len(tagNames) > 0 {
		// ハッキング情報テーブルと中間テーブルをハッキング情報IDで結合
		// 中間テーブルとタグテーブルをタグIDで結合
		query += `
			JOIN hacking_info_tags it ON hi.id = it.info_id
			JOIN tags t ON it.tag_id = t.id
			WHERE t.name IN (?)
		`
		// スライスに含まれるタグを持つハッキング情報を指定
		var err error
		query, args, err = sqlx.In(query, tagNames)
		if err != nil {
			return nil, fmt.Errorf("failed to expand IN clause: %w", err)
		}
		// すでに取得している情報のIDより過去の情報を取得
		if prevInfoID > 0 {
			query += " AND hi.id < ?"
			args = append(args, prevInfoID)
		}
	} else {
		// すでに取得している情報のIDより過去の情報を取得
		if prevInfoID > 0 {
			query += " WHERE hi.id < ?"
			args = append(args, prevInfoID)
		}
	}

	// ID順に整列、指定件数取得
	query += " ORDER BY hi.id DESC LIMIT ?"
	args = append(args, infoNumber)
	// データベースドライバに合わせてプレースホルダーを変換
	query = r.db.Rebind(query)

	// クエリ実行
	var infos []*entity.HackingInfo
	if err := r.db.SelectContext(ctx, &infos, query, args...); err != nil {
		return nil, fmt.Errorf("failed to select infos: %w", err)
	}

	// ハッキング情報が見つからなければ、処理を終了
	if len(infos) == 0 {
		return infos, nil
	}

	// 取得したハッキング情報IDに紐づく全てのタグを取得
	infoIDs := make([]int64, len(infos))
	for i, info := range infos {
		infoIDs[i] = info.ID
	}

	// タグテーブルに中間テーブルをタグIDで結合
	tagsQuery := `
		SELECT t.id, t.name, it.info_id
		FROM tags t
		JOIN hacking_info_tags it ON t.id = it.tag_id
		WHERE it.info_id IN (?)
	`

	// タグ取得用の構造体
	type infoTag struct {
		entity.Tag
		InfoID int64 `db:"info_id"`
	}
	var tags []infoTag

	// 取得したハッキング情報のタグを指定
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

	// 取得したタグをハッキング情報にマッピング
	tagsByInfoID := make(map[int64][]*entity.Tag)
	for _, t := range tags {
		tag := &entity.Tag{ID: t.ID, Name: t.Name}
		tagsByInfoID[t.InfoID] = append(tagsByInfoID[t.InfoID], tag)
	}

	// ハッキング情報のスライスにタグをセット
	for _, info := range infos {
		if associatedTags, ok := tagsByInfoID[info.ID]; ok {
			info.Tags = associatedTags
		}
	}

	return infos, nil
}

// 存在するすべてのタグを取得
func (r *hackingRepository) GetAllTags(ctx context.Context) ([]*entity.Tag, error) {
	var tags []*entity.Tag
	query := "SELECT id, name FROM tags ORDER BY name"
	if err := r.db.SelectContext(ctx, &tags, query); err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, nil
}

// 新しいハッキング情報と関連タグをトランザクション内で保存
func (r *hackingRepository) StoreInfo(ctx context.Context, info *entity.HackingInfo, tagNames []string) (int64, error) {
	// トランザクションを開始
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	// 関数を抜ける際にエラーがあればロールバック
	defer tx.Rollback()

	// ハッキング情報を保存するクエリ文を設定
	stmt, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO hacking_infos (protocol, network, amount, tx_hash, report_time)
		VALUES (:protocol, :network, :amount, :tx_hash, :report_time)
		RETURNING id
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare info statement: %w", err)
	}
	defer stmt.Close()

	var infoID int64
	// ハッキング情報を `hacking_infos` テーブルに保存
	// ハッキング情報のIDを取得
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

	// 中間テーブル `hacking_info_tags` にハッキング情報とタグの関連を保存
	for _, tagID := range tagIDs {
		_, err := tx.ExecContext(ctx, "INSERT INTO hacking_info_tags (info_id, tag_id) VALUES ($1, $2)", infoID, tagID)
		if err != nil {
			return 0, fmt.Errorf("failed to insert into hacking_info_tags: %w", err)
		}
	}

	// トランザクションをコミットして変更を確定
	return infoID, tx.Commit()
}