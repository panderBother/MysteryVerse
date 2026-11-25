package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewWritesStructuredJSON(t *testing.T) {
	var output bytes.Buffer
	logger := New("debug", &output)

	logger.Info("service started", "service", "go-api")

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
		t.Fatalf("log is not valid JSON: %v", err)
	}

	if record["msg"] != "service started" {
		t.Fatalf("msg = %v, want service started", record["msg"])
	}

	if !strings.Contains(output.String(), `"service":"go-api"`) {
		t.Fatalf("structured field missing from %s", output.String())
	}
}
