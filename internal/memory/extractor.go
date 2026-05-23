package memory

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const maxHunkContent = 12000

// Entry represents one file change within a commit.
type Entry struct {
	CommitHash    string
	Branch        string
	Author        string
	FilePath      string
	Intent        string
	What          string
	Why           string
	Kind          string
	Language      string
	Tags          []string
	RawDiff       string
	ContentBefore string
	ContentAfter  string
	Hunks         []Hunk
}

// Hunk represents a single @@ block in a diff.
type Hunk struct {
	LinesStart int
	LinesEnd   int
	Symbol     string
	ChangeType string
	Diff       string
}

var languageMap = map[string]string{
	".java": "java", ".kt": "kotlin", ".scala": "scala",
	".py": "python", ".go": "go", ".rs": "rust",
	".ts": "typescript", ".tsx": "typescript",
	".js": "javascript", ".jsx": "javascript",
	".cs": "csharp", ".rb": "ruby", ".php": "php",
	".c": "c", ".cpp": "cpp", ".h": "c",
	".sh": "shell", ".bash": "shell", ".zsh": "shell",
	".yaml": "yaml", ".yml": "yaml",
	".json": "json", ".toml": "toml",
	".sql": "sql", ".md": "markdown",
	".css": "css", ".scss": "scss",
}

var codeExts = map[string]bool{
	".java": true, ".py": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".go": true, ".rs": true, ".kt": true, ".cs": true, ".cpp": true, ".c": true,
	".h": true, ".swift": true, ".rb": true, ".php": true, ".scala": true,
	".sh": true, ".bash": true, ".zsh": true, ".sql": true, ".groovy": true,
}

var docExts = map[string]bool{
	".md": true, ".mdx": true, ".rst": true, ".adoc": true, ".txt": true,
}

var hunkHeaderRe = regexp.MustCompile(`@@\s+-\d+(?:,\d+)?\s+\+(\d+)(?:,(\d+))?\s+@@(.*)`)
var commitTypeRe = regexp.MustCompile(`^(\w+)(?:\(([\w/.\-]+)\))?:`)

// RepoName returns the current git repository folder name.
func RepoName() string {
	top := gitCmd("rev-parse", "--show-toplevel")
	if idx := strings.LastIndex(top, "/"); idx >= 0 {
		return top[idx+1:]
	}
	if top == "" {
		return "hoa-project"
	}
	return top
}

// Extract extracts all file-level entries for a given commit ref.
func Extract(ref string) (string, []Entry, error) {
	hash := gitCmd("log", "-1", "--format=%h", ref)
	if hash == "" {
		return "", nil, nil
	}
	author := gitCmd("log", "-1", "--format=%an", ref)
	intent := gitCmd("log", "-1", "--format=%s", ref)
	body := gitCmd("log", "-1", "--format=%b", ref)
	branch := resolveBranch(ref)

	what, why := parseBody(body)
	commitType, scope := parseCommitParts(intent)

	parent := gitCmd("rev-parse", "--verify", ref+"~1")
	if parent == "" {
		parent = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	}

	filesRaw := gitCmd("diff-tree", "--no-commit-id", "-r", "--name-only", ref)
	files := splitNonEmpty(filesRaw)

	entries := make([]Entry, 0, len(files))
	for _, file := range files {
		lang := detectLanguage(file)
		kind := memoryKind(file)

		diff := gitCmd("diff", parent, ref, "--", file)
		if diff == "" {
			diff = gitCmd("show", ref, "--", file)
		}

		hunks := parseHunks(diff)
		if len(hunks) == 0 {
			hunks = []Hunk{{LinesStart: 1, LinesEnd: 1, ChangeType: "modification"}}
		}

		ctype := overallChangeType(hunks)
		tags := buildTags(commitType, ctype, fileKind(file), scope)

		// Content before/after (truncated to avoid huge payloads)
		contentAfter := gitCmd("show", ref+":"+file)
		contentBefore := ""
		if parent != "4b825dc642cb6eb9a060e54bf8d69288fbee4904" {
			contentBefore = gitCmd("show", parent+":"+file)
		}
		const maxContent = 32000
		if len(diff) > maxContent {
			diff = diff[:maxContent]
		}
		if len(contentAfter) > maxContent {
			contentAfter = contentAfter[:maxContent]
		}
		if len(contentBefore) > maxContent {
			contentBefore = contentBefore[:maxContent]
		}

		entries = append(entries, Entry{
			CommitHash:    hash,
			Branch:        branch,
			Author:        author,
			FilePath:      file,
			Intent:        commitType,
			What:          fallback(what, intent),
			Why:           why,
			Kind:          kind,
			Language:      lang,
			Tags:          tags,
			RawDiff:       diff,
			ContentBefore: contentBefore,
			ContentAfter:  contentAfter,
			Hunks:         truncateHunks(hunks),
		})
	}
	return hash, entries, nil
}

// NeedsEnrichment returns true if the entry lacks proper intent/why (legacy commit).
func NeedsEnrichment(e Entry) bool {
	// No conventional commit type
	if e.Intent == "" {
		return true
	}
	// No explicit why
	if e.Why == "" {
		return true
	}
	return false
}

func parseBody(body string) (what, why string) {
	for _, line := range strings.Split(body, "\n") {
		switch {
		case strings.HasPrefix(line, "what:"):
			what = strings.TrimSpace(line[5:])
		case strings.HasPrefix(line, "why:"):
			why = strings.TrimSpace(line[4:])
		}
	}
	return
}

func parseCommitParts(intent string) (ctype, scope string) {
	m := commitTypeRe.FindStringSubmatch(intent)
	if m == nil {
		return "", ""
	}
	return m[1], m[2]
}

func parseHunks(diff string) []Hunk {
	lines := strings.Split(diff, "\n")
	var hunks []Hunk
	var cur *Hunk
	var curDiff []string

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			if cur != nil {
				cur.Diff = strings.Join(curDiff, "\n")
				cur.ChangeType = hunkChangeType(curDiff)
				hunks = append(hunks, *cur)
			}
			m := hunkHeaderRe.FindStringSubmatch(line)
			if m == nil {
				cur = nil
				continue
			}
			start := atoi(m[1])
			count := 1
			if m[2] != "" {
				count = atoi(m[2])
			}
			cur = &Hunk{
				LinesStart: start,
				LinesEnd:   start + max(count-1, 0),
				Symbol:     strings.TrimSpace(m[3]),
			}
			curDiff = nil
		} else if cur != nil {
			if (strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++")) ||
				(strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---")) {
				curDiff = append(curDiff, line)
			}
		}
	}
	if cur != nil {
		cur.Diff = strings.Join(curDiff, "\n")
		cur.ChangeType = hunkChangeType(curDiff)
		hunks = append(hunks, *cur)
	}
	return hunks
}

func hunkChangeType(lines []string) string {
	hasAdd, hasDel := false, false
	for _, l := range lines {
		if strings.HasPrefix(l, "+") {
			hasAdd = true
		} else if strings.HasPrefix(l, "-") {
			hasDel = true
		}
	}
	if hasAdd && hasDel {
		return "modification"
	}
	if hasAdd {
		return "addition"
	}
	return "deletion"
}

func overallChangeType(hunks []Hunk) string {
	hasAdd, hasDel := false, false
	for _, h := range hunks {
		switch h.ChangeType {
		case "addition":
			hasAdd = true
		case "deletion":
			hasDel = true
		case "modification":
			return "modification"
		}
	}
	if hasAdd && hasDel {
		return "modification"
	}
	if hasAdd {
		return "addition"
	}
	return "deletion"
}

func truncateHunks(hunks []Hunk) []Hunk {
	total := 0
	for i := range hunks {
		if total+len(hunks[i].Diff) > maxHunkContent {
			hunks[i].Diff = hunks[i].Diff[:max(maxHunkContent-total, 0)]
			return hunks[:i+1]
		}
		total += len(hunks[i].Diff)
	}
	return hunks
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if l, ok := languageMap[ext]; ok {
		return l
	}
	return "other"
}

func memoryKind(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	lower := strings.ToLower(path)
	if codeExts[ext] {
		return "code"
	}
	if docExts[ext] {
		return "doc"
	}
	if ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".toml" || ext == ".xml" {
		if strings.Contains(lower, "openapi") || strings.Contains(lower, "swagger") ||
			strings.Contains(lower, "docs/") || strings.Contains(lower, "doc/") {
			return "doc"
		}
	}
	return "config"
}

func fileKind(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	name := strings.ToLower(filepath.Base(path))
	switch {
	case ext == ".sh" || ext == ".bash" || ext == ".zsh":
		return "script"
	case codeExts[ext]:
		if strings.Contains(name, "test") || strings.Contains(name, "spec") {
			return "test"
		}
		return "source"
	case ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".env" || ext == ".ini":
		return "config"
	case ext == ".md" || ext == ".txt" || ext == ".rst":
		return "doc"
	case ext == ".css" || ext == ".scss":
		return "style"
	}
	return "other"
}

func resolveBranch(ref string) string {
	if ref == "HEAD" {
		if b := gitCmd("rev-parse", "--abbrev-ref", "HEAD"); b != "" {
			return b
		}
	}
	raw := gitCmd("branch", "--contains", ref, "--format=%(refname:short)")
	for _, b := range strings.Split(raw, "\n") {
		b = strings.TrimSpace(b)
		if b == "main" || b == "master" {
			return b
		}
	}
	if lines := splitNonEmpty(raw); len(lines) > 0 {
		return lines[0]
	}
	return "unknown"
}

func buildTags(commitType, changeType, kind, scope string) []string {
	var tags []string
	for _, t := range []string{commitType, changeType, kind, scope} {
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func fallback(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func gitCmd(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
