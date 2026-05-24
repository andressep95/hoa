package tool

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudcentinel/hoa/internal/api"
	"github.com/cloudcentinel/hoa/internal/memory"
)

// OracleQueryTool provides structured historical and analytical queries against
// Oracle 23ai — answering questions that semantic vector search cannot handle:
// temporal history, authorship, churn, coupling, documentation, and system health.
type OracleQueryTool struct {
	dsn    string
	apiKey string
}

func NewOracleQueryTool(dsn, apiKey string) *OracleQueryTool {
	return &OracleQueryTool{dsn: dsn, apiKey: apiKey}
}

func (t *OracleQueryTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name: "oracle_query",
		Description: `Structured queries against Oracle 23ai project history and knowledge base.
Use this for temporal, authorship, and analytical questions that search_memory (vector/semantic) cannot answer.

Available types and their params (req = required):

HISTORY
  file_history    file_path(req), since, until, author, intent, limit
                  → All commits that touched a file over time
  commit_detail   commit_hash(req)
                  → Every file changed in a commit, with diffs
  blame           symbol(req), file_path
                  → Which commit last changed a function/symbol
  symbol_history  symbol(req), file_path, limit
                  → Full evolution of a function across all commits

ACTIVITY
  author_activity author(req), since, until, intent, limit
                  → What a developer changed and when
  recent_commits  since, until, intent, kind, language, author, limit
                  → Recent changes with optional filters (intent: feat/fix/refactor/docs/test/chore)
  team_activity   since, until
                  → All contributors: commit counts and last active date

ANALYTICS
  hottest_files   since, until, limit
                  → Most frequently changed files (churn analysis)
  co_changes      file_path(req), limit
                  → Files that always change together with this file (coupling)

KNOWLEDGE BASE
  search_docs     query(req), doc_type, limit
                  → Semantic search in ADRs, runbooks, guides, and other docs
  list_docs       doc_type
                  → Catalog of all indexed documentation (doc_type: ADR/API_SPEC/RUNBOOK/GUIDE/README/CHANGELOG/ONBOARDING/DESIGN/OTHER)

SYSTEM
  list_feedback   scope
                  → All active feedback rules, optionally filtered by file scope
  enrichment_status  (no params)
                  → Health of the commit enrichment queue (PENDING/PROCESSING/DONE/FAILED)`,
		InputSchema: map[string]any{
			"type":        map[string]any{"type": "string", "description": "Query type — see tool description for full list."},
			"file_path":   map[string]any{"type": "string", "description": "File path (e.g. internal/agent/agent.go)."},
			"commit_hash": map[string]any{"type": "string", "description": "Git commit hash (full or short)."},
			"author":      map[string]any{"type": "string", "description": "Git author name (exact match)."},
			"symbol":      map[string]any{"type": "string", "description": "Function, struct, or method name (partial match supported)."},
			"intent":      map[string]any{"type": "string", "description": "Commit intent: feat, fix, refactor, docs, test, chore, perf, style."},
			"kind":        map[string]any{"type": "string", "description": "File kind: code, doc, config."},
			"language":    map[string]any{"type": "string", "description": "Programming language (e.g. go, python, typescript)."},
			"since":       map[string]any{"type": "string", "description": "Start date ISO 8601 (e.g. 2026-04-01). Inclusive."},
			"until":       map[string]any{"type": "string", "description": "End date ISO 8601 (e.g. 2026-05-01). Exclusive."},
			"query":       map[string]any{"type": "string", "description": "Natural language query for semantic document search."},
			"doc_type":    map[string]any{"type": "string", "description": "Document type filter: ADR, API_SPEC, RUNBOOK, GUIDE, README, CHANGELOG, ONBOARDING, DESIGN, OTHER."},
			"scope":       map[string]any{"type": "string", "description": "File path pattern to filter feedback rules by scope."},
			"limit":       map[string]any{"type": "integer", "description": "Max results (default 10, max 50)."},
		},
		Required: []string{"type"},
	}
}

type oracleQueryInput struct {
	Type       string `json:"type"`
	FilePath   string `json:"file_path"`
	CommitHash string `json:"commit_hash"`
	Author     string `json:"author"`
	Symbol     string `json:"symbol"`
	Intent     string `json:"intent"`
	Kind       string `json:"kind"`
	Language   string `json:"language"`
	Since      string `json:"since"`
	Until      string `json:"until"`
	Query      string `json:"query"`
	DocType    string `json:"doc_type"`
	Scope      string `json:"scope"`
	Limit      int    `json:"limit"`
}

func (t *OracleQueryTool) Execute(_ context.Context, input string) (string, bool) {
	var in oracleQueryInput
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	if in.Limit <= 0 {
		in.Limit = 10
	}
	if in.Limit > 50 {
		in.Limit = 50
	}

	mc, err := memory.NewClient(t.dsn, t.apiKey)
	if err != nil {
		return fmt.Sprintf("oracle connect: %v", err), true
	}
	defer mc.Close()

	db := mc.DB()
	pid := mc.ProjectID()

	switch in.Type {
	case "file_history":
		return oqFileHistory(db, pid, in)
	case "commit_detail":
		return oqCommitDetail(db, pid, in)
	case "blame":
		return oqBlame(db, pid, in)
	case "symbol_history":
		return oqSymbolHistory(db, pid, in)
	case "author_activity":
		return oqAuthorActivity(db, pid, in)
	case "recent_commits":
		return oqRecentCommits(db, pid, in)
	case "hottest_files":
		return oqHottestFiles(db, pid, in)
	case "co_changes":
		return oqCoChanges(db, pid, in)
	case "team_activity":
		return oqTeamActivity(db, pid, in)
	case "search_docs":
		return oqSearchDocs(db, pid, in)
	case "list_docs":
		return oqListDocs(db, pid, in)
	case "list_feedback":
		return oqListFeedback(db, pid, in)
	case "enrichment_status":
		return oqEnrichmentStatus(db, pid)
	default:
		return fmt.Sprintf("unknown type: %q. Valid: file_history, commit_detail, blame, symbol_history, author_activity, recent_commits, hottest_files, co_changes, team_activity, search_docs, list_docs, list_feedback, enrichment_status", in.Type), true
	}
}

// ── HISTORY ──────────────────────────────────────────────────────────────────

func oqFileHistory(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	if in.FilePath == "" {
		return "file_history requires file_path", true
	}
	q, args := newOQ(pid)
	q.str("file_path = :%d", in.FilePath)
	q.str("author = :%d", in.Author)
	q.str("intent = :%d", in.Intent)
	q.date("created_at >= TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Since)
	q.date("created_at < TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Until)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT commit_hash, branch, author, intent,
		       DBMS_LOB.SUBSTR(what,300,1), DBMS_LOB.SUBSTR(why,200,1), created_at
		FROM HOA.memory_changes
		WHERE project_id = HEXTORAW(:1)%s
		ORDER BY created_at DESC
		FETCH FIRST %d ROWS ONLY`, q.where(), in.Limit), args...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	fmt.Fprintf(&sb, "## File history: %s\n\n", in.FilePath)
	n := 0
	for rows.Next() {
		var hash, branch, author, intent, what string
		var why *string
		var ts string
		if err := rows.Scan(&hash, &branch, &author, &intent, &what, &why, &ts); err != nil {
			continue
		}
		n++
		abbrev := hash
		if len(abbrev) > 7 {
			abbrev = abbrev[:7]
		}
		fmt.Fprintf(&sb, "### %d. %s — %s by %s [%s]\n", n, abbrev, fmtTS(ts), author, intent)
		fmt.Fprintf(&sb, "**What:** %s\n", what)
		if why != nil && *why != "" {
			fmt.Fprintf(&sb, "**Why:** %s\n", *why)
		}
		sb.WriteString("\n")
	}
	if n == 0 {
		return fmt.Sprintf("No history found for %s with the given filters.", in.FilePath), false
	}
	return sb.String(), false
}

func oqCommitDetail(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	if in.CommitHash == "" {
		return "commit_detail requires commit_hash", true
	}
	rows, err := db.Query(`
		SELECT mc.file_path, mc.intent, mc.kind, mc.author, mc.branch,
		       DBMS_LOB.SUBSTR(mc.what,500,1),
		       DBMS_LOB.SUBSTR(mc.why,300,1),
		       DBMS_LOB.SUBSTR(mc.raw_diff,4000,1),
		       mc.created_at
		FROM HOA.memory_changes mc
		WHERE mc.project_id = HEXTORAW(:1)
		  AND mc.commit_hash LIKE :2
		ORDER BY mc.file_path`,
		pid, in.CommitHash+"%")
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	n := 0
	var commitMeta string
	for rows.Next() {
		var filePath, intent, kind, author, branch, what, ts string
		var why, rawDiff *string
		if err := rows.Scan(&filePath, &intent, &kind, &author, &branch, &what, &why, &rawDiff, &ts); err != nil {
			continue
		}
		if n == 0 {
			commitMeta = fmt.Sprintf("## Commit %s\n**Author:** %s · **Branch:** %s · **Date:** %s\n\n", in.CommitHash, author, branch, fmtTS(ts))
		}
		n++
		fmt.Fprintf(&sb, "### %s [%s/%s]\n", filePath, kind, intent)
		fmt.Fprintf(&sb, "**What:** %s\n", what)
		if why != nil && *why != "" {
			fmt.Fprintf(&sb, "**Why:** %s\n", *why)
		}
		if rawDiff != nil && *rawDiff != "" {
			fmt.Fprintf(&sb, "```diff\n%s\n```\n", strings.TrimRight(*rawDiff, "\n"))
		}
		sb.WriteString("\n")
	}
	if n == 0 {
		return fmt.Sprintf("No commit found matching %q.", in.CommitHash), false
	}
	return commitMeta + sb.String(), false
}

func oqBlame(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	if in.Symbol == "" {
		return "blame requires symbol", true
	}
	q, args := newOQ(pid)
	q.str("mc.file_path = :%d", in.FilePath)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT mc.commit_hash, mc.author, mc.file_path,
		       h.symbol, h.change_type,
		       DBMS_LOB.SUBSTR(h.hunk_diff,2000,1),
		       DBMS_LOB.SUBSTR(mc.what,300,1),
		       mc.created_at
		FROM HOA.memory_changes mc
		JOIN HOA.memory_change_hunks h ON h.memory_change_id = mc.id
		WHERE mc.project_id = HEXTORAW(:1)
		  AND UPPER(h.symbol) LIKE UPPER('%%'||:%d||'%%')%s
		ORDER BY mc.created_at DESC
		FETCH FIRST 5 ROWS ONLY`, q.nextPos(), q.where()),
		append(args, in.Symbol)...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Blame: %q\n\n", in.Symbol)
	n := 0
	for rows.Next() {
		var hash, author, filePath, symbol, changeType, what, ts string
		var hunkDiff *string
		if err := rows.Scan(&hash, &author, &filePath, &symbol, &changeType, &hunkDiff, &what, &ts); err != nil {
			continue
		}
		n++
		abbrev := hash
		if len(abbrev) > 7 {
			abbrev = abbrev[:7]
		}
		fmt.Fprintf(&sb, "### %d. %s in %s — %s by %s (%s)\n", n, abbrev, filePath, fmtTS(ts), author, changeType)
		fmt.Fprintf(&sb, "**Symbol:** %s\n**What:** %s\n", symbol, what)
		if hunkDiff != nil && *hunkDiff != "" {
			fmt.Fprintf(&sb, "```diff\n%s\n```\n", strings.TrimRight(*hunkDiff, "\n"))
		}
		sb.WriteString("\n")
	}
	if n == 0 {
		return fmt.Sprintf("No commits found that touched symbol %q.", in.Symbol), false
	}
	return sb.String(), false
}

func oqSymbolHistory(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	if in.Symbol == "" {
		return "symbol_history requires symbol", true
	}
	q, args := newOQ(pid)
	q.str("mc.file_path = :%d", in.FilePath)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT mc.commit_hash, mc.author, mc.file_path,
		       h.symbol, h.change_type,
		       DBMS_LOB.SUBSTR(mc.what,300,1),
		       mc.created_at
		FROM HOA.memory_changes mc
		JOIN HOA.memory_change_hunks h ON h.memory_change_id = mc.id
		WHERE mc.project_id = HEXTORAW(:1)
		  AND UPPER(h.symbol) LIKE UPPER('%%'||:%d||'%%')%s
		ORDER BY mc.created_at ASC
		FETCH FIRST %d ROWS ONLY`, q.nextPos(), q.where(), in.Limit),
		append(args, in.Symbol)...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Symbol history: %q\n\n", in.Symbol)
	n := 0
	for rows.Next() {
		var hash, author, filePath, symbol, changeType, what, ts string
		if err := rows.Scan(&hash, &author, &filePath, &symbol, &changeType, &what, &ts); err != nil {
			continue
		}
		n++
		abbrev := hash
		if len(abbrev) > 7 {
			abbrev = abbrev[:7]
		}
		fmt.Fprintf(&sb, "%d. `%s` — %s by %s [%s] in %s\n   %s\n\n", n, abbrev, fmtTS(ts), author, changeType, filePath, what)
	}
	if n == 0 {
		return fmt.Sprintf("No history found for symbol %q.", in.Symbol), false
	}
	return sb.String(), false
}

// ── ACTIVITY ─────────────────────────────────────────────────────────────────

func oqAuthorActivity(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	if in.Author == "" {
		return "author_activity requires author", true
	}
	q, args := newOQ(pid)
	q.str("author = :%d", in.Author)
	q.str("intent = :%d", in.Intent)
	q.date("created_at >= TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Since)
	q.date("created_at < TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Until)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT commit_hash, file_path, intent,
		       DBMS_LOB.SUBSTR(what,300,1), created_at
		FROM HOA.memory_changes
		WHERE project_id = HEXTORAW(:1)%s
		ORDER BY created_at DESC
		FETCH FIRST %d ROWS ONLY`, q.where(), in.Limit), args...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Activity: %s\n\n", in.Author)
	n := 0
	for rows.Next() {
		var hash, filePath, intent, what, ts string
		if err := rows.Scan(&hash, &filePath, &intent, &what, &ts); err != nil {
			continue
		}
		n++
		abbrev := hash
		if len(abbrev) > 7 {
			abbrev = abbrev[:7]
		}
		fmt.Fprintf(&sb, "%d. `%s` %s [%s] — %s\n   %s\n\n", n, abbrev, filePath, intent, fmtTS(ts), what)
	}
	if n == 0 {
		return fmt.Sprintf("No activity found for %q with the given filters.", in.Author), false
	}
	return sb.String(), false
}

func oqRecentCommits(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	q, args := newOQ(pid)
	q.str("author = :%d", in.Author)
	q.str("intent = :%d", in.Intent)
	q.str("kind = :%d", in.Kind)
	q.str("language = :%d", in.Language)
	q.date("created_at >= TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Since)
	q.date("created_at < TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Until)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT commit_hash, file_path, author, intent, kind,
		       DBMS_LOB.SUBSTR(what,250,1), created_at
		FROM HOA.memory_changes
		WHERE project_id = HEXTORAW(:1)%s
		ORDER BY created_at DESC
		FETCH FIRST %d ROWS ONLY`, q.where(), in.Limit), args...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("## Recent commits\n\n")
	n := 0
	for rows.Next() {
		var hash, filePath, author, intent, kind, what, ts string
		if err := rows.Scan(&hash, &filePath, &author, &intent, &kind, &what, &ts); err != nil {
			continue
		}
		n++
		abbrev := hash
		if len(abbrev) > 7 {
			abbrev = abbrev[:7]
		}
		fmt.Fprintf(&sb, "%d. `%s` %s — %s [%s/%s] by %s\n   %s\n\n", n, abbrev, fmtTS(ts), filePath, kind, intent, author, what)
	}
	if n == 0 {
		return "No commits found with the given filters.", false
	}
	return sb.String(), false
}

func oqTeamActivity(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	q, args := newOQ(pid)
	q.date("created_at >= TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Since)
	q.date("created_at < TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Until)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT author,
		       COUNT(*) as file_changes,
		       COUNT(DISTINCT commit_hash) as commits,
		       MAX(created_at) as last_active
		FROM HOA.memory_changes
		WHERE project_id = HEXTORAW(:1)%s
		GROUP BY author
		ORDER BY commits DESC`, q.where()), args...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("## Team activity\n\n")
	sb.WriteString("| Author | Commits | File changes | Last active |\n")
	sb.WriteString("|--------|---------|--------------|-------------|\n")
	n := 0
	for rows.Next() {
		var author, lastActive string
		var fileChanges, commits int
		if err := rows.Scan(&author, &fileChanges, &commits, &lastActive); err != nil {
			continue
		}
		n++
		fmt.Fprintf(&sb, "| %s | %d | %d | %s |\n", author, commits, fileChanges, fmtTS(lastActive))
	}
	if n == 0 {
		return "No activity found.", false
	}
	return sb.String(), false
}

// ── ANALYTICS ────────────────────────────────────────────────────────────────

func oqHottestFiles(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	q, args := newOQ(pid)
	q.date("created_at >= TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Since)
	q.date("created_at < TO_TIMESTAMP_TZ(:%d,'YYYY-MM-DD')", in.Until)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT file_path,
		       COUNT(*) as change_count,
		       COUNT(DISTINCT commit_hash) as commits,
		       MAX(created_at) as last_changed
		FROM HOA.memory_changes
		WHERE project_id = HEXTORAW(:1)%s
		GROUP BY file_path
		ORDER BY change_count DESC
		FETCH FIRST %d ROWS ONLY`, q.where(), in.Limit), args...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("## Hottest files (churn)\n\n")
	sb.WriteString("| File | Changes | Commits | Last changed |\n")
	sb.WriteString("|------|---------|---------|-------------|\n")
	n := 0
	for rows.Next() {
		var filePath, lastChanged string
		var changes, commits int
		if err := rows.Scan(&filePath, &changes, &commits, &lastChanged); err != nil {
			continue
		}
		n++
		fmt.Fprintf(&sb, "| %s | %d | %d | %s |\n", filePath, changes, commits, fmtTS(lastChanged))
	}
	if n == 0 {
		return "No data found.", false
	}
	return sb.String(), false
}

func oqCoChanges(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	if in.FilePath == "" {
		return "co_changes requires file_path", true
	}
	rows, err := db.Query(fmt.Sprintf(`
		SELECT other.file_path, COUNT(*) as times_together
		FROM HOA.memory_changes target
		JOIN HOA.memory_changes other
		  ON target.commit_hash = other.commit_hash
		 AND target.project_id = other.project_id
		 AND other.file_path != target.file_path
		WHERE target.project_id = HEXTORAW(:1)
		  AND target.file_path = :2
		GROUP BY other.file_path
		ORDER BY times_together DESC
		FETCH FIRST %d ROWS ONLY`, in.Limit),
		pid, in.FilePath)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Files that change with: %s\n\n", in.FilePath)
	sb.WriteString("| File | Times together |\n")
	sb.WriteString("|------|---------------|\n")
	n := 0
	for rows.Next() {
		var filePath string
		var times int
		if err := rows.Scan(&filePath, &times); err != nil {
			continue
		}
		n++
		fmt.Fprintf(&sb, "| %s | %d |\n", filePath, times)
	}
	if n == 0 {
		return fmt.Sprintf("%s has never changed together with any other file.", in.FilePath), false
	}
	return sb.String(), false
}

// ── KNOWLEDGE BASE ────────────────────────────────────────────────────────────

func oqSearchDocs(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	if in.Query == "" {
		return "search_docs requires query", true
	}
	q, args := newOQ(pid)
	q.str("d.doc_type = :%d", in.DocType)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT d.source_path, d.title, d.doc_type,
		       ds.heading,
		       DBMS_LOB.SUBSTR(ds.content,1000,1),
		       VECTOR_DISTANCE(ds.embedding, VECTOR_EMBEDDING(HOA_EMBED_MODEL USING :%d AS data), COSINE) AS score
		FROM HOA.document_sections ds
		JOIN HOA.documents d ON ds.document_id = d.id
		WHERE d.project_id = HEXTORAW(:1)
		  AND ds.embedding IS NOT NULL%s
		ORDER BY score ASC
		FETCH FIRST %d ROWS ONLY`, q.nextPos(), q.where(), in.Limit),
		append(args, in.Query)...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Doc search: %q\n\n", in.Query)
	n := 0
	for rows.Next() {
		var sourcePath, title, docType, heading, content string
		var score float64
		if err := rows.Scan(&sourcePath, &title, &docType, &heading, &content, &score); err != nil {
			continue
		}
		n++
		fmt.Fprintf(&sb, "### [%d] %s › %s (%.2f) [%s]\n", n, title, heading, score, docType)
		fmt.Fprintf(&sb, "**Source:** %s\n%s\n\n", sourcePath, content)
	}
	if n == 0 {
		return "No documents found. The documents table may not be populated yet.", false
	}
	return sb.String(), false
}

func oqListDocs(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	q, args := newOQ(pid)
	q.str("doc_type = :%d", in.DocType)

	rows, err := db.Query(fmt.Sprintf(`
		SELECT source_path, title, doc_type, stale, indexed_at
		FROM HOA.documents
		WHERE project_id = HEXTORAW(:1)%s
		ORDER BY doc_type, title`, q.where()), args...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("## Indexed documents\n\n")
	sb.WriteString("| Type | Title | Path | Stale | Indexed |\n")
	sb.WriteString("|------|-------|------|-------|---------|\n")
	n := 0
	for rows.Next() {
		var sourcePath, title, docType, indexedAt string
		var stale int
		if err := rows.Scan(&sourcePath, &title, &docType, &stale, &indexedAt); err != nil {
			continue
		}
		n++
		staleLabel := "✓"
		if stale == 1 {
			staleLabel = "⚠ stale"
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s |\n", docType, title, sourcePath, staleLabel, fmtTS(indexedAt))
	}
	if n == 0 {
		return "No documents indexed. Populate the HOA.documents table to use this feature.", false
	}
	return sb.String(), false
}

// ── SYSTEM ────────────────────────────────────────────────────────────────────

func oqListFeedback(db *sql.DB, pid string, in oracleQueryInput) (string, bool) {
	q, args := newOQ(pid)
	if in.Scope != "" {
		q.str("scope LIKE '%%'||:%d||'%%'", in.Scope)
	}

	rows, err := db.Query(fmt.Sprintf(`
		SELECT rule, why, scope, created_at
		FROM HOA.feedback_rules
		WHERE project_id = HEXTORAW(:1)
		  AND active = 1%s
		ORDER BY created_at DESC`, q.where()), args...)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("## Active feedback rules\n\n")
	n := 0
	for rows.Next() {
		var rule, ts string
		var why, scope *string
		if err := rows.Scan(&rule, &why, &scope, &ts); err != nil {
			continue
		}
		n++
		fmt.Fprintf(&sb, "%d. **%s**\n", n, rule)
		if why != nil && *why != "" {
			fmt.Fprintf(&sb, "   Why: %s\n", *why)
		}
		if scope != nil && *scope != "" {
			fmt.Fprintf(&sb, "   Scope: `%s`\n", *scope)
		}
		fmt.Fprintf(&sb, "   Added: %s\n\n", fmtTS(ts))
	}
	if n == 0 {
		return "No active feedback rules.", false
	}
	return sb.String(), false
}

func oqEnrichmentStatus(db *sql.DB, pid string) (string, bool) {
	rows, err := db.Query(`
		SELECT eq.status, COUNT(*) as cnt, MAX(eq.created_at) as latest
		FROM HOA.enrichment_queue eq
		JOIN HOA.memory_changes mc ON mc.id = eq.memory_change_id
		WHERE mc.project_id = HEXTORAW(:1)
		GROUP BY eq.status
		ORDER BY eq.status`, pid)
	if err != nil {
		return fmt.Sprintf("query error: %v", err), true
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("## Enrichment queue status\n\n")
	sb.WriteString("| Status | Count | Latest |\n")
	sb.WriteString("|--------|-------|--------|\n")
	n := 0
	for rows.Next() {
		var status, latest string
		var cnt int
		if err := rows.Scan(&status, &cnt, &latest); err != nil {
			continue
		}
		n++
		fmt.Fprintf(&sb, "| %s | %d | %s |\n", status, cnt, fmtTS(latest))
	}
	if n == 0 {
		return "No enrichment queue entries for this project.", false
	}
	return sb.String(), false
}

// ── helpers ───────────────────────────────────────────────────────────────────

// oq is a WHERE-clause builder for positional Oracle bindings.
// Position 1 is always reserved for project_id (HEXTORAW(:1)).
type oq struct {
	pos  int
	cond []string
	args []any
}

func newOQ(pid string) (*oq, []any) {
	return &oq{pos: 1}, []any{pid}
}

func (q *oq) nextPos() int {
	q.pos++
	return q.pos
}

func (q *oq) str(clause, val string) {
	if val == "" {
		return
	}
	q.pos++
	q.cond = append(q.cond, fmt.Sprintf(clause, q.pos))
	q.args = append(q.args, val)
}

func (q *oq) date(clause, val string) {
	if val == "" {
		return
	}
	q.pos++
	q.cond = append(q.cond, fmt.Sprintf(clause, q.pos))
	q.args = append(q.args, val)
}

func (q *oq) where() string {
	if len(q.cond) == 0 {
		return ""
	}
	return "\n  AND " + strings.Join(q.cond, "\n  AND ")
}

// fmtTS trims Oracle timestamp strings to a readable date+time.
func fmtTS(ts string) string {
	if len(ts) >= 16 {
		return ts[:16]
	}
	return ts
}

// Ensure OracleQueryTool satisfies the Tool interface at compile time.
var _ interface {
	Definition() api.ToolDef
	Execute(context.Context, string) (string, bool)
} = (*OracleQueryTool)(nil)
