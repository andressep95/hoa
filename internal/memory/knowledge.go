package memory

import (
	"fmt"
	"strings"
)

// FetchProjectKnowledge builds a per-file index of what Oracle knows.
// One row per unique file_path using the latest indexed commit for that file.
// Used by the LLM as a routing signal: if the file is listed, call search_memory;
// if not listed, use filesystem tools directly.
func FetchProjectKnowledge(client *Client) (string, error) {
	rows, err := client.db.Query(`
		SELECT mc.file_path,
		       mc.intent,
		       DBMS_LOB.SUBSTR(mc.what, 120, 1)
		FROM HOA.memory_changes mc
		INNER JOIN (
		    SELECT file_path, MAX(created_at) AS latest
		    FROM HOA.memory_changes
		    WHERE project_id = HEXTORAW(:1)
		    GROUP BY file_path
		) lmc ON mc.file_path = lmc.file_path AND mc.created_at = lmc.latest
		WHERE mc.project_id = HEXTORAW(:1)
		ORDER BY mc.file_path
		FETCH FIRST 150 ROWS ONLY`,
		client.projectID, client.projectID,
	)
	if err != nil {
		return "", fmt.Errorf("fetch project knowledge: %w", err)
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("<project_knowledge>\n")
	sb.WriteString("Files Oracle has indexed (call search_memory for details on any of these):\n")
	count := 0
	for rows.Next() {
		var filePath, what string
		var intent *string
		if err := rows.Scan(&filePath, &intent, &what); err != nil {
			return "", err
		}
		what = strings.TrimSpace(what)
		tag := ""
		if intent != nil && *intent != "" {
			tag = "[" + *intent + "] "
		}
		fmt.Fprintf(&sb, "- %s %s\n", filePath, tag+what)
		count++
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	sb.WriteString("</project_knowledge>")
	if count == 0 {
		return "", nil
	}
	return sb.String(), nil
}
