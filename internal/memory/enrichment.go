package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
)

const enrichBatchSize = 5

// LLMEnricher is the function signature for LLM calls (maps to ctx.AgentSend).
type LLMEnricher func(prompt string) (string, error)

// EnrichmentProcessor drains the enrichment queue in background.
type EnrichmentProcessor struct {
	client  *Client
	llm     LLMEnricher
	running atomic.Bool
	done    chan struct{}
}

// NewEnrichmentProcessor creates a processor tied to a DB client and LLM function.
func NewEnrichmentProcessor(client *Client, llm LLMEnricher) *EnrichmentProcessor {
	return &EnrichmentProcessor{
		client: client,
		llm:    llm,
		done:   make(chan struct{}, 1),
	}
}

// Trigger starts async processing if not already running. Non-blocking.
func (ep *EnrichmentProcessor) Trigger() {
	if ep.running.CompareAndSwap(false, true) {
		go ep.drainQueue()
	}
}

// Wait blocks until the processor finishes draining.
func (ep *EnrichmentProcessor) Wait() {
	<-ep.done
}

func (ep *EnrichmentProcessor) drainQueue() {
	defer func() {
		ep.running.Store(false)
		select {
		case ep.done <- struct{}{}:
		default:
		}
	}()

	for {
		tasks, err := ep.fetchBatch()
		if err != nil || len(tasks) == 0 {
			return
		}
		for _, t := range tasks {
			ep.processOne(t)
		}
	}
}

type enrichmentTask struct {
	taskID         string
	memoryChangeID string
	what           string
	filePath       string
	rawDiff        sql.NullString
}

func (ep *EnrichmentProcessor) fetchBatch() ([]enrichmentTask, error) {
	rows, err := ep.client.db.Query(`
		SELECT RAWTOHEX(eq.id), RAWTOHEX(eq.memory_change_id),
		       mc.what, mc.file_path, mc.raw_diff
		FROM HOA.enrichment_queue eq
		JOIN HOA.memory_changes mc ON mc.id = eq.memory_change_id
		WHERE eq.status = 'PENDING'
		ORDER BY eq.created_at
		FETCH FIRST :1 ROWS ONLY`,
		enrichBatchSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []enrichmentTask
	for rows.Next() {
		var t enrichmentTask
		if err := rows.Scan(&t.taskID, &t.memoryChangeID, &t.what, &t.filePath, &t.rawDiff); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

const enrichPrompt = `You are a git commit analyzer. Given the commit message, file path, and diff,
infer structured semantic fields. Respond ONLY with valid JSON, no markdown fences.
{
  "intent": "feat|fix|refactor|perf|docs|test|chore|ci|style|sec",
  "what": "<one sentence: what the code does now>",
  "why": "<one sentence: why this change was necessary>"
}

Commit message: %s
File: %s
Diff:
%s`

type enrichResult struct {
	Intent string `json:"intent"`
	What   string `json:"what"`
	Why    string `json:"why"`
}

func (ep *EnrichmentProcessor) processOne(t enrichmentTask) {
	// Mark as processing
	ep.client.db.Exec(
		"UPDATE HOA.enrichment_queue SET status = 'PROCESSING' WHERE id = HEXTORAW(:1)",
		t.taskID,
	)

	diff := ""
	if t.rawDiff.Valid {
		diff = t.rawDiff.String
		if len(diff) > 2000 {
			diff = diff[:2000]
		}
	}

	prompt := fmt.Sprintf(enrichPrompt, t.what, t.filePath, diff)
	response, err := ep.llm(prompt)
	if err != nil {
		ep.markFailed(t.taskID, err.Error())
		return
	}

	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result enrichResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		ep.markFailed(t.taskID, "invalid JSON: "+err.Error())
		return
	}

	// Update memory_change with enriched data
	_, err = ep.client.db.Exec(`
		UPDATE HOA.memory_changes
		SET intent = :1, what = :2, why = :3
		WHERE id = HEXTORAW(:4)`,
		result.Intent, result.What, result.Why, t.memoryChangeID,
	)
	if err != nil {
		ep.markFailed(t.taskID, err.Error())
		return
	}

	// Mark done
	ep.client.db.Exec(
		"UPDATE HOA.enrichment_queue SET status = 'DONE', processed_at = SYSTIMESTAMP WHERE id = HEXTORAW(:1)",
		t.taskID,
	)
}

func (ep *EnrichmentProcessor) markFailed(taskID, errMsg string) {
	if len(errMsg) > 900 {
		errMsg = errMsg[:900]
	}
	ep.client.db.Exec(`
		UPDATE HOA.enrichment_queue
		SET status = CASE WHEN attempts >= 2 THEN 'FAILED' ELSE 'PENDING' END,
		    attempts = attempts + 1,
		    last_error = :1
		WHERE id = HEXTORAW(:2)`,
		errMsg, taskID,
	)
}
