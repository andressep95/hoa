package memory

import "fmt"

// FeedbackRule represents a user correction/guidance stored in Oracle.
type FeedbackRule struct {
	ID    string
	Rule  string
	Why   string
	Scope string
}

// SaveFeedback inserts a new feedback rule.
func (c *Client) SaveFeedback(rule, why, scope string) error {
	id := newUUID()
	_, err := c.db.Exec(`INSERT INTO HOA.feedback_rules (id, project_id, rule, why, scope)
		VALUES (HEXTORAW(:1), HEXTORAW(:2), :3, :4, :5)`,
		id, c.projectID, rule, why, nullStr(scope),
	)
	return err
}

// SupersedeFeedback marks an old rule as superseded by a new one.
func (c *Client) SupersedeFeedback(oldID, newRule, newWhy, newScope string) error {
	newID := newUUID()
	_, err := c.db.Exec(`INSERT INTO HOA.feedback_rules (id, project_id, rule, why, scope)
		VALUES (HEXTORAW(:1), HEXTORAW(:2), :3, :4, :5)`,
		newID, c.projectID, newRule, newWhy, nullStr(newScope),
	)
	if err != nil {
		return err
	}
	_, err = c.db.Exec(`UPDATE HOA.feedback_rules SET active = 0, superseded_by = HEXTORAW(:1) WHERE id = HEXTORAW(:2)`,
		newID, oldID,
	)
	return err
}

// SearchFeedback finds active feedback rules relevant to the query.
func (c *Client) SearchFeedback(query string, limit int) ([]FeedbackRule, error) {
	rows, err := c.db.Query(`
		SELECT RAWTOHEX(id), rule, why, scope
		FROM HOA.feedback_rules
		WHERE project_id = HEXTORAW(:1)
		  AND active = 1
		  AND embedding IS NOT NULL
		ORDER BY VECTOR_DISTANCE(embedding, VECTOR_EMBEDDING(HOA_EMBED_MODEL USING :2 AS data), COSINE) ASC
		FETCH FIRST :3 ROWS ONLY`,
		c.projectID, query, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search feedback: %w", err)
	}
	defer rows.Close()

	var rules []FeedbackRule
	for rows.Next() {
		var r FeedbackRule
		var why, scope *string
		if err := rows.Scan(&r.ID, &r.Rule, &why, &scope); err != nil {
			return nil, err
		}
		if why != nil {
			r.Why = *why
		}
		if scope != nil {
			r.Scope = *scope
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// FormatFeedback formats feedback rules as context for LLM injection.
func FormatFeedback(rules []FeedbackRule) string {
	if len(rules) == 0 {
		return ""
	}
	ctx := "<feedback_rules>\n"
	for _, r := range rules {
		ctx += fmt.Sprintf("- %s", r.Rule)
		if r.Why != "" {
			ctx += fmt.Sprintf(" (why: %s)", r.Why)
		}
		if r.Scope != "" {
			ctx += fmt.Sprintf(" [scope: %s]", r.Scope)
		}
		ctx += "\n"
	}
	ctx += "</feedback_rules>"
	return ctx
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
