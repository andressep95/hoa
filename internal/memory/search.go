package memory

import (
	"fmt"
	"strings"
)

// contentAfterLimit is the number of characters fetched from content_after.
// Files larger than this will be flagged as truncated in the formatted output.
const contentAfterLimit = 6000

// SearchResult represents a relevant memory entry found by semantic search.
type SearchResult struct {
	CommitHash   string
	FilePath     string
	What         string
	Why          string
	Kind         string
	Score        float64
	ContentAfter string // actual file content after this change (may be truncated)
}

// Search performs semantic search against memory_changes using the user's prompt.
// Oracle generates the embedding of the query on-the-fly and compares via VECTOR_DISTANCE.
// Returns up to `limit` results ordered by relevance (ascending cosine distance).
// Results with score > 0.55 (low cosine similarity) are excluded.
func Search(client *Client, query string, limit int) ([]SearchResult, error) {
	rows, err := client.db.Query(`
		SELECT commit_hash, file_path,
		       DBMS_LOB.SUBSTR(what, 800, 1),
		       DBMS_LOB.SUBSTR(why, 800, 1),
		       kind,
		       DBMS_LOB.SUBSTR(content_after, 6000, 1),
		       VECTOR_DISTANCE(embedding, VECTOR_EMBEDDING(HOA_EMBED_MODEL USING :1 AS data), COSINE) AS score
		FROM HOA.memory_changes
		WHERE project_id = HEXTORAW(:2)
		  AND embedding IS NOT NULL
		  AND VECTOR_DISTANCE(embedding, VECTOR_EMBEDDING(HOA_EMBED_MODEL USING :1 AS data), COSINE) < 0.55
		ORDER BY score ASC
		FETCH FIRST :3 ROWS ONLY`,
		query, client.projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("memory search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var why, contentAfter *string
		if err := rows.Scan(&r.CommitHash, &r.FilePath, &r.What, &why, &r.Kind, &contentAfter, &r.Score); err != nil {
			return nil, err
		}
		if why != nil {
			r.Why = *why
		}
		if contentAfter != nil {
			r.ContentAfter = *contentAfter
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// FormatContext formats search results as structured blocks for LLM consumption.
// Each result includes the file path, relevance score, what changed, why, and
// the actual file content after the change so the LLM has full context without
// needing to call read_file.
func FormatContext(results []SearchResult) string {
	if len(results) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<project_memory>\n")
	for i, r := range results {
		hash := r.CommitHash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		fmt.Fprintf(&sb, "### [%d] %s · commit %s · %s · relevance %.2f\n", i+1, r.FilePath, hash, r.Kind, r.Score)
		fmt.Fprintf(&sb, "**What:** %s\n", r.What)
		if r.Why != "" {
			fmt.Fprintf(&sb, "**Why:** %s\n", r.Why)
		}
		if r.ContentAfter != "" {
			label := "**Current content (complete):**"
			if len(r.ContentAfter) >= contentAfterLimit-100 {
				label = "**Current content (truncated — call read_file for the full version):**"
			}
			sb.WriteString(label + "\n```\n")
			sb.WriteString(strings.TrimRight(r.ContentAfter, "\n"))
			sb.WriteString("\n```\n")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("</project_memory>")
	return sb.String()
}
