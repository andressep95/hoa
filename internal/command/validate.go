package command

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// type(scope): description
	headerRe = regexp.MustCompile(`^(feat|fix|refactor|perf|sec|test|docs|chore|ci|style)(\([a-zA-Z0-9_./-]+\))?: .{1,72}$`)

	// Types that require what/why/breaking in body
	richTypes = map[string]bool{
		"feat": true, "fix": true, "refactor": true, "perf": true, "sec": true,
	}
)

// ValidateCommitMsg checks a commit message against the skill spec.
// Returns nil if valid, or a list of violations.
func ValidateCommitMsg(msg string) []string {
	lines := strings.Split(strings.TrimSpace(msg), "\n")
	if len(lines) == 0 {
		return []string{"Mensaje vacío."}
	}

	var errs []string
	header := lines[0]

	// Validate header format
	if !headerRe.MatchString(header) {
		errs = append(errs, fmt.Sprintf("Header no cumple formato: type(scope): description (max 72 chars)\n    Recibido: %q", header))
	}

	// Check trailing period
	if strings.HasSuffix(header, ".") {
		errs = append(errs, "No usar punto final en el header.")
	}

	// Extract type to check if body fields are required
	m := regexp.MustCompile(`^(\w+)`).FindStringSubmatch(header)
	if m == nil {
		return errs
	}
	commitType := m[1]

	if !richTypes[commitType] {
		return errs // chore, docs, test, ci, style don't need body fields
	}

	// For feat/fix/refactor/perf/sec: require what, why, breaking
	body := strings.Join(lines[1:], "\n")
	if !strings.Contains(body, "what:") {
		errs = append(errs, "Falta campo 'what:' en el body (requerido para "+commitType+").")
	}
	if !strings.Contains(body, "why:") {
		errs = append(errs, "Falta campo 'why:' en el body (requerido para "+commitType+").")
	}
	if !strings.Contains(body, "breaking:") {
		errs = append(errs, "Falta campo 'breaking:' en el body (requerido para "+commitType+").")
	}

	// Validate what/why are not empty
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "what:") && strings.TrimSpace(strings.TrimPrefix(trimmed, "what:")) == "" {
			errs = append(errs, "Campo 'what:' no puede estar vacío.")
		}
		if strings.HasPrefix(trimmed, "why:") && strings.TrimSpace(strings.TrimPrefix(trimmed, "why:")) == "" {
			errs = append(errs, "Campo 'why:' no puede estar vacío.")
		}
	}

	return errs
}
