package rsync

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type HistoryReadResult struct {
	Entries []Result
	Skipped int
}

// LoadHistory returns at most limit entries, ordered newest first.
func LoadHistory(stateDirectory string, limit int) (HistoryReadResult, error) {
	if limit <= 0 {
		return HistoryReadResult{}, nil
	}
	file, err := os.Open(filepath.Join(stateDirectory, "history.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return HistoryReadResult{}, nil
	}
	if err != nil {
		return HistoryReadResult{}, err
	}
	defer file.Close()

	result := HistoryReadResult{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var entry Result
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			result.Skipped++
			continue
		}
		result.Entries = append(result.Entries, entry)
		if len(result.Entries) > limit {
			copy(result.Entries, result.Entries[len(result.Entries)-limit:])
			result.Entries = result.Entries[:limit]
		}
	}
	if err := scanner.Err(); err != nil {
		return HistoryReadResult{}, err
	}
	for left, right := 0, len(result.Entries)-1; left < right; left, right = left+1, right-1 {
		result.Entries[left], result.Entries[right] = result.Entries[right], result.Entries[left]
	}
	return result, nil
}
