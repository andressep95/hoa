package memory

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"
)

// maxFileContentBytes is the character limit when fetching content_after by file path.
// Large enough for most source files; truncated files are flagged in the result.
const maxFileContentBytes = 30000

// MaxFileContentBytes exposes the truncation limit for callers (e.g. tool output messages).
func MaxFileContentBytes() int { return maxFileContentBytes }

// FileContent holds the result of a direct file lookup in Oracle.
type FileContent struct {
	Content    string
	CommitHash string
	Truncated  bool
}

// GetLatestFileContent returns the most recently indexed content_after for a file path.
// Returns found=false (no error) when the file is not in Oracle for this project.
func (c *Client) GetLatestFileContent(filePath string) (fc FileContent, found bool, err error) {
	var content string
	err = c.db.QueryRow(`
		SELECT DBMS_LOB.SUBSTR(content_after, :1, 1), commit_hash
		FROM HOA.memory_changes
		WHERE project_id = HEXTORAW(:2)
		  AND file_path = :3
		  AND content_after IS NOT NULL
		ORDER BY created_at DESC
		FETCH FIRST 1 ROW ONLY`,
		maxFileContentBytes, c.projectID, filePath,
	).Scan(&content, &fc.CommitHash)
	if err == sql.ErrNoRows {
		return FileContent{}, false, nil
	}
	if err != nil {
		return FileContent{}, false, err
	}
	fc.Content = content
	fc.Truncated = len(content) >= maxFileContentBytes-100
	return fc, true, nil
}

// Client connects directly to Oracle 23ai for memory operations.
type Client struct {
	db        *sql.DB
	projectID string
}

// NewClient opens a connection to Oracle and resolves the project by API key.
func NewClient(dsn, apiKey string) (*Client, error) {
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("oracle connect: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("oracle ping: %w", err)
	}

	var projectID string
	err = db.QueryRow("SELECT RAWTOHEX(id) FROM HOA.projects WHERE api_key = :1", apiKey).Scan(&projectID)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("API key no encontrada: %w", err)
	}

	return &Client{db: db, projectID: projectID}, nil
}

// ConnectDSN opens a connection to Oracle without resolving a project (for setup).
func ConnectDSN(dsn string) (*Client, error) {
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("oracle connect: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("oracle ping: %w", err)
	}
	return &Client{db: db}, nil
}

// CreateProject inserts a new project and returns the generated API key.
func (c *Client) CreateProject(name string) (string, error) {
	apiKey := generateAPIKey()
	id := newUUID()
	_, err := c.db.Exec(
		`INSERT INTO HOA.projects (id, api_key, name) VALUES (HEXTORAW(:1), :2, :3)`,
		id, apiKey, name,
	)
	if err != nil {
		return "", fmt.Errorf("crear proyecto: %w", err)
	}
	c.projectID = id
	return apiKey, nil
}

// ResolveProject sets the project by API key on an existing connection.
func (c *Client) ResolveProject(apiKey string) error {
	var projectID string
	err := c.db.QueryRow("SELECT RAWTOHEX(id) FROM HOA.projects WHERE api_key = :1", apiKey).Scan(&projectID)
	if err != nil {
		return fmt.Errorf("API key no encontrada: %w", err)
	}
	c.projectID = projectID
	return nil
}

// Close releases the database connection.
func (c *Client) Close() error { return c.db.Close() }

// Ping verifies the connection is alive.
func (c *Client) Ping() error { return c.db.Ping() }

// ProjectID returns the resolved project UUID.
func (c *Client) ProjectID() string { return c.projectID }

// DB returns the underlying connection for advanced structured queries.
func (c *Client) DB() *sql.DB { return c.db }

// GetIndexedCommits returns all commit hashes already stored for this project.
func (c *Client) GetIndexedCommits() (map[string]bool, error) {
	rows, err := c.db.Query(
		"SELECT DISTINCT commit_hash FROM HOA.memory_changes WHERE project_id = HEXTORAW(:1)",
		c.projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := make(map[string]bool)
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		set[h] = true
	}
	return set, rows.Err()
}

// BatchInsertResult holds counters from a batch insert.
type BatchInsertResult struct {
	Inserted         int
	Skipped          int
	EnrichmentQueued int
}

// BatchInsert inserts entries into memory_changes + memory_change_hunks.
// Skips entries whose (commit_hash, file_path) already exist.
func (c *Client) BatchInsert(entries []Entry) (BatchInsertResult, error) {
	if len(entries) == 0 {
		return BatchInsertResult{}, nil
	}

	tx, err := c.db.Begin()
	if err != nil {
		return BatchInsertResult{}, err
	}
	defer tx.Rollback()

	var res BatchInsertResult
	for _, e := range entries {
		changeID := newUUID()
		tags := strings.Join(e.Tags, ",")

		_, err := tx.Exec(`INSERT INTO HOA.memory_changes
			(id, project_id, commit_hash, branch, author, file_path, kind, intent, what, why, language, tags, raw_diff, content_before, content_after)
			VALUES (HEXTORAW(:1), HEXTORAW(:2), :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15)`,
			changeID, c.projectID, e.CommitHash, e.Branch, e.Author, e.FilePath,
			e.Kind, e.Intent,
			go_ora.Clob{String: e.What, Valid: e.What != ""},
			go_ora.Clob{String: e.Why, Valid: e.Why != ""},
			e.Language, tags,
			go_ora.Clob{String: e.RawDiff, Valid: e.RawDiff != ""},
			go_ora.Clob{String: e.ContentBefore, Valid: e.ContentBefore != ""},
			go_ora.Clob{String: e.ContentAfter, Valid: e.ContentAfter != ""},
		)
		if err != nil {
			if isUniqueViolation(err) {
				res.Skipped++
				continue
			}
			return res, fmt.Errorf("insert change %s:%s: %w", e.CommitHash, e.FilePath, err)
		}
		res.Inserted++

		for _, h := range e.Hunks {
			hunkID := newUUID()
			changeType := h.ChangeType
			if changeType == "" {
				changeType = "modification"
			}
			hunkDiff := h.Diff
			if hunkDiff == "" {
				hunkDiff = " "
			}
			_, err := tx.Exec(`INSERT INTO HOA.memory_change_hunks
				(id, memory_change_id, lines_start, lines_end, symbol, change_type, hunk_diff)
				VALUES (HEXTORAW(:1), HEXTORAW(:2), :3, :4, :5, :6, :7)`,
				hunkID, changeID, h.LinesStart, h.LinesEnd, h.Symbol, changeType,
				go_ora.Clob{String: hunkDiff, Valid: true},
			)
			if err != nil {
				return res, fmt.Errorf("insert hunk: %w", err)
			}
		}

		if NeedsEnrichment(e) {
			enrichID := newUUID()
			_, err := tx.Exec(`INSERT INTO HOA.enrichment_queue (id, memory_change_id)
				VALUES (HEXTORAW(:1), HEXTORAW(:2))`,
				enrichID, changeID,
			)
			if err != nil {
				return res, fmt.Errorf("enqueue enrichment: %w", err)
			}
			res.EnrichmentQueued++
		}
	}

	if err := tx.Commit(); err != nil {
		return BatchInsertResult{}, err
	}
	return res, nil
}

// CountIndexed returns total indexed changes and pending enrichment for this project.
func (c *Client) CountIndexed() (indexed int, pendingEnrich int, err error) {
	err = c.db.QueryRow(
		"SELECT COUNT(*) FROM HOA.memory_changes WHERE project_id = HEXTORAW(:1)",
		c.projectID,
	).Scan(&indexed)
	if err != nil {
		return
	}
	err = c.db.QueryRow(`SELECT COUNT(*) FROM HOA.enrichment_queue eq
		JOIN HOA.memory_changes mc ON mc.id = eq.memory_change_id
		WHERE mc.project_id = HEXTORAW(:1) AND eq.status = 'PENDING'`,
		c.projectID,
	).Scan(&pendingEnrich)
	return
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "ORA-00001")
}

func newUUID() string {
	b := make([]byte, 16)
	crand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func generateAPIKey() string {
	b := make([]byte, 24)
	crand.Read(b)
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := range b {
		b[i] = chars[b[i]%36]
	}
	return "hoa_" + string(b)
}
