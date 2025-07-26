package database

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

// Series 剧集信息
type Series struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	URL             string     `json:"url"`
	History         []string   `json:"history"`     // 历史集数 ["S01E01", "S01E02", ...]
	Current         string     `json:"current"`     // 当前更新到的集数 "S03E02"
	IsWatched       bool       `json:"is_watched"`  // 当前集数是否已观看
	IsTracking      bool       `json:"is_tracking"` // 是否启用追踪
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"` // History, Current 更新才算
	CrawlerLastSeen *time.Time `json:"crawler_last_seen"`
}

// Settings 全局配置
type Settings struct {
	CrawlerStartTime string `json:"crawler_start_time"`
	CrawlerEndTime   string `json:"crawler_end_time"`
	SlackWebhookURL  string `json:"slack_webhook_url"`
}

// FetchTask 爬虫任务
type FetchTask struct {
	URLs []string `json:"tasks"`
}

// FetchResult 爬虫结果
type FetchResult struct {
	Name   string   `json:"name"`
	Update string   `json:"update"`
	URL    string   `json:"url"`
	Series []string `json:"series"`
}

// FetchCallback 爬虫回调
type FetchCallback struct {
	Tasks   []string      `json:"tasks"`
	Results []FetchResult `json:"results"`
	Status  int           `json:"status"`
	Message string        `json:"message"`
}

func NewDatabase(dbPath string) (*Database, error) {
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) CreateTables() error {
	// 创建剧集表
	createSeriesTable := `
	CREATE TABLE IF NOT EXISTS series (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT UNIQUE NOT NULL,
		history TEXT NOT NULL DEFAULT '[]',
		current TEXT NOT NULL DEFAULT '',
		is_watched BOOLEAN NOT NULL DEFAULT 0,
		is_tracking BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := d.db.Exec(createSeriesTable)

	if err != nil {
		return err
	}

	if exists, err := d.columnExists("series", "crawler_last_seen"); err == nil && !exists {
		_, err = d.db.Exec("ALTER TABLE series ADD COLUMN crawler_last_seen DATETIME")
		if err != nil {
			return err
		}
	}

	// 创建全局配置表
	createSettingsTable := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);`
	_, err = d.db.Exec(createSettingsTable)
	if err != nil {
		return err
	}

	return err
}

func (d *Database) columnExists(tableName, columnName string) (bool, error) {
	rows, err := d.db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, nil
}

// 获取所有剧集
func (d *Database) GetAllSeries() ([]Series, error) {
	rows, err := d.db.Query(`
		SELECT id, name, url, history, current, is_watched, is_tracking, created_at, updated_at, crawler_last_seen
		FROM series
		ORDER BY is_tracking DESC, updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var series []Series
	for rows.Next() {
		var s Series
		var historyJSON string
		var crawlerLastSeen sql.NullTime
		err := rows.Scan(
			&s.ID, &s.Name, &s.URL, &historyJSON, &s.Current,
			&s.IsWatched, &s.IsTracking, &s.CreatedAt, &s.UpdatedAt,
			&crawlerLastSeen,
		)
		if err != nil {
			return nil, err
		}

		// 解析历史集数 JSON
		if err := json.Unmarshal([]byte(historyJSON), &s.History); err != nil {
			return nil, err
		}

		if crawlerLastSeen.Valid {
			s.CrawlerLastSeen = &crawlerLastSeen.Time
		}

		series = append(series, s)
	}

	return series, nil
}

// 创建剧集
func (d *Database) CreateSeries(name, url string) (*Series, error) {
	historyJSON, _ := json.Marshal([]string{})

	result, err := d.db.Exec(`
		INSERT INTO series (name, url, history, current, is_watched, is_tracking)
		VALUES (?, ?, ?, '', 0, 1)
	`, name, url, historyJSON)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return d.GetSeriesByID(id)
}

// 根据ID获取剧集
func (d *Database) GetSeriesByID(id int64) (*Series, error) {
	var s Series
	var historyJSON string
	var crawlerLastSeen sql.NullTime
	err := d.db.QueryRow(`
		SELECT id, name, url, history, current, is_watched, is_tracking, created_at, updated_at, crawler_last_seen
		FROM series WHERE id = ?
	`, id).Scan(
		&s.ID, &s.Name, &s.URL, &historyJSON, &s.Current,
		&s.IsWatched, &s.IsTracking, &s.CreatedAt, &s.UpdatedAt,
		&crawlerLastSeen,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(historyJSON), &s.History); err != nil {
		return nil, err
	}

	if crawlerLastSeen.Valid {
		s.CrawlerLastSeen = &crawlerLastSeen.Time
	}

	return &s, nil
}

// 更新剧集
func (d *Database) UpdateSeries(id int64, name, url string) error {
	_, err := d.db.Exec(`
		UPDATE series 
		SET name = ?, url = ?
		WHERE id = ?
	`, name, url, id)
	return err
}

// 删除剧集
func (d *Database) DeleteSeries(id int64) error {
	_, err := d.db.Exec("DELETE FROM series WHERE id = ?", id)
	return err
}

// 标记为已观看
func (d *Database) MarkAsWatched(id int64) error {
	_, err := d.db.Exec(`
		UPDATE series 
		SET is_watched = 1
		WHERE id = ?
	`, id)
	return err
}

// 标记为未观看
func (d *Database) MarkAsUnwatched(id int64) error {
	_, err := d.db.Exec(`
		UPDATE series 
		SET is_watched = 0
		WHERE id = ?
	`, id)
	return err
}

// 切换追踪状态
func (d *Database) ToggleTracking(id int64) error {
	_, err := d.db.Exec(`
		UPDATE series 
		SET is_tracking = CASE WHEN is_tracking = 1 THEN 0 ELSE 1 END
		WHERE id = ?
	`, id)
	return err
}

// 更新剧集信息（爬虫回调使用）
func (d *Database) UpdateSeriesInfo(url string, current string, series []string) error {
	historyJSON, err := json.Marshal(series)
	if err != nil {
		return err
	}

	// 有新集数时，自动重置为未看状态，因为新集数还没看
	_, err = d.db.Exec(`
		UPDATE series 
		SET current = ?, history = ?, is_watched = 0, updated_at = CURRENT_TIMESTAMP,
			crawler_last_seen = CURRENT_TIMESTAMP
		WHERE url = ?
	`, current, historyJSON, url)
	return err
}

// 更新剧集爬虫最后更新时间
func (d *Database) UpdateSeriesCrawlerLastSeen(url string, lastSeen time.Time) error {
	_, err := d.db.Exec(`
		UPDATE series 
		SET crawler_last_seen = ?
		WHERE url = ?
	`, lastSeen, url)
	return err
}

// 获取所有启用的剧集URL（爬虫任务使用）
func (d *Database) GetAllTrackingURLs() ([]string, error) {
	rows, err := d.db.Query("SELECT url FROM series WHERE is_tracking = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	return urls, nil
}

// 根据URL获取剧集信息
func (d *Database) GetSeriesByURL(url string) (*Series, error) {
	var s Series
	var historyJSON string
	var crawlerLastSeen sql.NullTime
	err := d.db.QueryRow(`
		SELECT id, name, url, history, current, is_watched, is_tracking, created_at, updated_at, crawler_last_seen
		FROM series WHERE url = ?
	`, url).Scan(
		&s.ID, &s.Name, &s.URL, &historyJSON, &s.Current,
		&s.IsWatched, &s.IsTracking, &s.CreatedAt, &s.UpdatedAt,
		&crawlerLastSeen,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(historyJSON), &s.History); err != nil {
		return nil, err
	}

	if crawlerLastSeen.Valid {
		s.CrawlerLastSeen = &crawlerLastSeen.Time
	}

	return &s, nil
}

// 清空剧集历史和当前进度
func (d *Database) ClearSeriesHistory(id int64) error {
	emptyHistory, _ := json.Marshal([]string{})
	_, err := d.db.Exec(`
		UPDATE series 
		SET history = ?, current = '', updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, emptyHistory, id)
	return err
}

// GetSettings 获取全局配置
func (d *Database) GetSettings() (*Settings, error) {
	settings := &Settings{}
	rows, err := d.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		switch key {
		case "crawler_start_time":
			settings.CrawlerStartTime = value
		case "crawler_end_time":
			settings.CrawlerEndTime = value
		case "slack_webhook_url":
			settings.SlackWebhookURL = value
		}
	}
	return settings, nil
}

// UpdateSettings 更新全局配置
func (d *Database) UpdateSettings(settings *Settings) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec("crawler_start_time", settings.CrawlerStartTime); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := stmt.Exec("crawler_end_time", settings.CrawlerEndTime); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := stmt.Exec("slack_webhook_url", settings.SlackWebhookURL); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
