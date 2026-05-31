package cli

import (
	"encoding/json"

	"github.com/shhac/agent-postmark/internal/output"
)

type evidenceRecord map[string]any

func entityRecord(object string, id any, data any) evidenceRecord {
	record := evidenceRecord{"type": "entity", "object": object, "data": data}
	if id != nil && id != "" {
		record["id"] = id
	}
	return record
}

func findingRecord(severity, summary string, data map[string]any) evidenceRecord {
	record := evidenceRecord{"type": "finding", "severity": severity, "summary": summary}
	if len(data) > 0 {
		record["data"] = data
	}
	return record
}

func nextCommandRecord(command, reason string) evidenceRecord {
	return evidenceRecord{"type": "next_command", "command": command, "reason": reason}
}

func writeEvidence(records []evidenceRecord) error {
	writer := output.NewNDJSONWriter(output.Stdout())
	for _, record := range records {
		if err := writer.WriteItem(record); err != nil {
			return err
		}
	}
	return nil
}

func rawObject(raw json.RawMessage) map[string]any {
	var out map[string]any
	_ = json.Unmarshal(redactRaw(raw), &out)
	if out == nil {
		return map[string]any{}
	}
	return out
}

func rawEnvelopeList(raw json.RawMessage, field string) []json.RawMessage {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	var rows []json.RawMessage
	_ = json.Unmarshal(payload[field], &rows)
	return rows
}

func rawTotal(raw json.RawMessage) int {
	var payload struct {
		TotalCount int `json:"TotalCount"`
	}
	_ = json.Unmarshal(raw, &payload)
	return payload.TotalCount
}
