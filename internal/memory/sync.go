package memory

import "fmt"

const batchSize = 25

// SyncResult holds the outcome of a sync operation.
type SyncResult struct {
	Total    int
	Inserted int
	Skipped  int
	Enriched int
}

// Sync compares local git history against Oracle and inserts missing commits.
// Enrichment runs concurrently — as entries are inserted and enqueued,
// the processor drains the queue calling the LLM in parallel.
func Sync(client *Client, branch string, llm LLMEnricher) (SyncResult, error) {
	indexed, err := client.GetIndexedCommits()
	if err != nil {
		return SyncResult{}, fmt.Errorf("fetching indexed commits: %w", err)
	}

	allCommits := splitNonEmpty(gitCmd("log", branch, "--reverse", "--format=%H"))

	var pending []string
	for _, full := range allCommits {
		short := full[:7]
		if !indexed[short] && !indexed[full] {
			pending = append(pending, full)
		}
	}

	if len(pending) == 0 {
		return SyncResult{Total: len(allCommits)}, nil
	}

	// Start enrichment processor in parallel
	var ep *EnrichmentProcessor
	if llm != nil {
		ep = NewEnrichmentProcessor(client, llm)
	}

	var res SyncResult
	res.Total = len(allCommits)

	var batch []Entry
	for _, ref := range pending {
		_, entries, err := Extract(ref)
		if err != nil || len(entries) == 0 {
			continue
		}
		batch = append(batch, entries...)

		if len(batch) >= batchSize {
			r, err := client.BatchInsert(batch)
			if err != nil {
				return res, err
			}
			res.Inserted += r.Inserted
			res.Skipped += r.Skipped
			res.Enriched += r.EnrichmentQueued

			// Trigger enrichment after each batch that queued tasks
			if r.EnrichmentQueued > 0 && ep != nil {
				ep.Trigger()
			}
			batch = batch[:0]
		}
	}

	// Flush remaining
	if len(batch) > 0 {
		r, err := client.BatchInsert(batch)
		if err != nil {
			return res, err
		}
		res.Inserted += r.Inserted
		res.Skipped += r.Skipped
		res.Enriched += r.EnrichmentQueued
		if r.EnrichmentQueued > 0 && ep != nil {
			ep.Trigger()
		}
	}

	// Wait for enrichment to finish draining
	if ep != nil && res.Enriched > 0 {
		ep.Wait()
	}

	return res, nil
}

// SyncOne extracts and inserts a single commit (used post-commit).
// Also triggers enrichment if needed.
func SyncOne(client *Client, ref string, llm LLMEnricher) (BatchInsertResult, error) {
	_, entries, err := Extract(ref)
	if err != nil {
		return BatchInsertResult{}, err
	}
	if len(entries) == 0 {
		return BatchInsertResult{}, nil
	}

	result, err := client.BatchInsert(entries)
	if err != nil {
		return result, err
	}

	if result.EnrichmentQueued > 0 && llm != nil {
		ep := NewEnrichmentProcessor(client, llm)
		ep.Trigger()
		ep.Wait()
	}

	return result, nil
}
