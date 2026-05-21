package claudejsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	jsonlInitialBuf = 64 * 1024
	jsonlMaxLine    = 4 * 1024 * 1024
)

type jsonlEntry struct {
	Type       string          `json:"type"`
	Timestamp  string          `json:"timestamp"`
	RawMessage json.RawMessage `json:"message"`

	msg *jsonlMessage
}

type jsonlMessage struct {
	RawContent  json.RawMessage `json:"content"`
	Content     []jsonlContent  `json:"-"`
	TextContent string          `json:"-"`
}

type jsonlContent struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (e *jsonlEntry) message() *jsonlMessage {
	if e.msg != nil {
		return e.msg
	}
	if len(e.RawMessage) == 0 || e.RawMessage[0] == 'n' {
		return nil
	}
	var m jsonlMessage
	if json.Unmarshal(e.RawMessage, &m) != nil {
		return nil
	}
	m.parseContent()
	e.msg = &m
	return e.msg
}

func (m *jsonlMessage) parseContent() {
	if len(m.RawContent) == 0 {
		return
	}
	switch m.RawContent[0] {
	case '"':
		_ = json.Unmarshal(m.RawContent, &m.TextContent)
	case '[':
		_ = json.Unmarshal(m.RawContent, &m.Content)
	}
}

func scanJSONL(path string) (scanResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return scanResult{}, fmt.Errorf("claudejsonl: open %q: %w", path, err)
	}
	defer f.Close()

	return scanReader(f)
}

func scanReader(r io.Reader) (scanResult, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, jsonlInitialBuf), jsonlMaxLine)

	var result scanResult
	var lastEntry *jsonlEntry
	var prevTimestamp time.Time

	for scanner.Scan() {
		var e jsonlEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, e.Timestamp)

		if e.Type == "summary" && !prevTimestamp.IsZero() {
			result.LastSummaryAt = prevTimestamp
		}

		if !ts.IsZero() {
			prevTimestamp = ts
		}

		if e.Type != "user" && e.Type != "assistant" {
			continue
		}

		msg := e.message()
		if msg == nil {
			continue
		}

		if e.Type == "assistant" && !ts.IsZero() {
			for _, c := range msg.Content {
				if c.Type == "tool_use" && c.Name != "" {
					result.LastTool = toolEvent{Name: c.Name, Timestamp: ts}
				}
			}
		}

		cp := e
		lastEntry = &cp
		if !ts.IsZero() {
			result.LastActivity = ts
		}
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("claudejsonl: scan: %w", err)
	}

	if lastEntry != nil {
		result.Status, result.CurrentTool = determineStatus(lastEntry)
	}
	return result, nil
}

func determineStatus(e *jsonlEntry) (jsonlStatus, string) {
	msg := e.message()
	if msg == nil {
		return statusUnknown, ""
	}
	switch e.Type {
	case "assistant":
		for _, c := range msg.Content {
			if c.Type == "tool_use" {
				return statusExecutingTool, c.Name
			}
		}
		return statusWaitingForUser, ""
	case "user":
		if isToolResult(msg) {
			return statusProcessingResult, ""
		}
		return statusThinking, ""
	}
	return statusUnknown, ""
}

func isToolResult(m *jsonlMessage) bool {
	for _, c := range m.Content {
		if c.Type == "tool_result" {
			return true
		}
	}
	return false
}
