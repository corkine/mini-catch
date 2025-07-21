package database

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

// Series 剧集信息
type Series struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	History    []string  `json:"history"`     // 历史集数 ["S01E01", "S01E02", ...]
	Current    string    `json:"current"`     // 当前更新到的集数 "S03E02"
	IsWatched  bool      `json:"is_watched"`  // 当前集数是否已观看
	IsTracking bool      `json:"is_tracking"` // 是否启用追踪
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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
	return err
}

// 获取所有剧集
func (d *Database) GetAllSeries() ([]Series, error) {
	rows, err := d.db.Query(`
		SELECT id, name, url, history, current, is_watched, is_tracking, created_at, updated_at
		FROM series
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var series []Series
	for rows.Next() {
		var s Series
		var historyJSON string
		err := rows.Scan(
			&s.ID, &s.Name, &s.URL, &historyJSON, &s.Current,
			&s.IsWatched, &s.IsTracking, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析历史集数 JSON
		if err := json.Unmarshal([]byte(historyJSON), &s.History); err != nil {
			return nil, err
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

	err := d.db.QueryRow(`
		SELECT id, name, url, history, current, is_watched, is_tracking, created_at, updated_at
		FROM series WHERE id = ?
	`, id).Scan(
		&s.ID, &s.Name, &s.URL, &historyJSON, &s.Current,
		&s.IsWatched, &s.IsTracking, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(historyJSON), &s.History); err != nil {
		return nil, err
	}

	return &s, nil
}

// 更新剧集
func (d *Database) UpdateSeries(id int64, name, url string) error {
	_, err := d.db.Exec(`
		UPDATE series 
		SET name = ?, url = ?, updated_at = CURRENT_TIMESTAMP
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
		SET is_watched = 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// 标记为未观看
func (d *Database) MarkAsUnwatched(id int64) error {
	_, err := d.db.Exec(`
		UPDATE series 
		SET is_watched = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// 切换追踪状态
func (d *Database) ToggleTracking(id int64) error {
	_, err := d.db.Exec(`
		UPDATE series 
		SET is_tracking = CASE WHEN is_tracking = 1 THEN 0 ELSE 1 END,
		    updated_at = CURRENT_TIMESTAMP
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
		SET current = ?, history = ?, is_watched = 0, updated_at = CURRENT_TIMESTAMP
		WHERE url = ?
	`, current, historyJSON, url)
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

	err := d.db.QueryRow(`
		SELECT id, name, url, history, current, is_watched, is_tracking, created_at, updated_at
		FROM series WHERE url = ?
	`, url).Scan(
		&s.ID, &s.Name, &s.URL, &historyJSON, &s.Current,
		&s.IsWatched, &s.IsTracking, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(historyJSON), &s.History); err != nil {
		return nil, err
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
