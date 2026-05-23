package memory

import "fmt"

// SearchResult represents a relevant memory entry found by semantic search.
type SearchResult struct {
	CommitHash string
	FilePath   string
	What       string
	Why        string
	Kind       string
	Score      float64
}

// Search performs semantic search against memory_changes using the user's prompt.
// Oracle generates the embedding of the query on-the-fly and compares via VECTOR_DISTANCE.
// Returns up to `limit` results. If score > threshold, they're irrelevant and excluded.
func Search(client *Client, query string, limit int) ([]SearchResult, error) {
	rows, err := client.db.Query(`
		SELECT commit_hash, file_path,
		       DBMS_LOB.SUBSTR(what, 200, 1),
		       DBMS_LOB.SUBSTR(why, 200, 1),
		       kind,
		       VECTOR_DISTANCE(embedding, VECTOR_EMBEDDING(HOA_EMBED_MODEL USING :1 AS data), COSINE) AS score
		FROM HOA.memory_changes
		WHERE project_id = HEXTORAW(:2)
		  AND embedding IS NOT NULL
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
		var why *string
		if err := rows.Scan(&r.CommitHash, &r.FilePath, &r.What, &why, &r.Kind, &r.Score); err != nil {
			return nil, err
		}
		if why != nil {
			r.Why = *why
		}
		// Filter out irrelevant results (cosine distance > 0.7 means low similarity)
		if r.Score > 0.7 {
			break
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// FormatContext formats search results as context string for LLM injection.
func FormatContext(results []SearchResult) string {
	if len(results) == 0 {
		return ""
	}
	ctx := "<project_memory>\n"
	for _, r := range results {
		ctx += fmt.Sprintf("- %s (%s) [%s]: %s", r.FilePath, r.CommitHash, r.Kind, r.What)
		if r.Why != "" {
			ctx += " — " + r.Why
		}
		ctx += "\n"
	}
	ctx += "</project_memory>"
	return ctx
}
