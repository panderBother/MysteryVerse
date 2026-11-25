package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readExample(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "..", "packages", "contracts", "examples", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example %s: %v", name, err)
	}

	return content
}

func TestWebSocketEnvelopeRoundTrip(t *testing.T) {
	var envelope WebSocketEnvelope
	if err := json.Unmarshal(readExample(t, "ws-game-envelope.json"), &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if envelope.GameID == nil {
		t.Fatal("GameID = nil, want game-scoped ID")
	}

	if envelope.Stream != WebSocketStreamGame {
		t.Fatalf("Stream = %q, want %q", envelope.Stream, WebSocketStreamGame)
	}

	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var roundTrip WebSocketEnvelope
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}

	if roundTrip.EventID != envelope.EventID || roundTrip.Seq != envelope.Seq {
		t.Fatal("round-trip envelope changed identity fields")
	}
}

func TestGeneratedModelsDeserializeCoreExamples(t *testing.T) {
	var apiError ApiErrorResponse
	if err := json.Unmarshal(readExample(t, "api-error.json"), &apiError); err != nil {
		t.Fatalf("unmarshal API error: %v", err)
	}
	if apiError.Error.Code != BusinessErrorCodeSTATEVERSIONCONFLICT {
		t.Fatalf("error code = %q", apiError.Error.Code)
	}

	var memoryCandidate MemoryCandidate
	if err := json.Unmarshal(readExample(t, "memory-candidate.json"), &memoryCandidate); err != nil {
		t.Fatalf("unmarshal MemoryCandidate: %v", err)
	}
	if memoryCandidate.AuthorityClass != AuthorityClassFact {
		t.Fatalf("authority class = %q", memoryCandidate.AuthorityClass)
	}

	var script Script
	if err := json.Unmarshal(readExample(t, "script.json"), &script); err != nil {
		t.Fatalf("unmarshal Script: %v", err)
	}
	if script.SchemaVersion != "1.0.0" {
		t.Fatalf("script schema version = %q", script.SchemaVersion)
	}

	var stage Stage
	if err := json.Unmarshal(readExample(t, "stage.json"), &stage); err != nil {
		t.Fatalf("unmarshal Stage: %v", err)
	}
	if stage.PhaseType != StagePhaseTypeSearch || stage.SequenceNo != 3 {
		t.Fatalf("stage = %#v", stage)
	}

	var asset ScriptAsset
	if err := json.Unmarshal(readExample(t, "script-asset.json"), &asset); err != nil {
		t.Fatalf("unmarshal ScriptAsset: %v", err)
	}
	if asset.AssetKey != "cover" || asset.Visibility != VisibilityScopePublic {
		t.Fatalf("asset = %#v", asset)
	}

	var validation ScriptValidationReport
	if err := json.Unmarshal(readExample(t, "script-validation-report.json"), &validation); err != nil {
		t.Fatalf("unmarshal ScriptValidationReport: %v", err)
	}
	if validation.Status != ScriptValidationReportStatusWarning ||
		len(validation.Issues) != 1 {
		t.Fatalf("validation report = %#v", validation)
	}

	var agentRequest AgentRequest
	if err := json.Unmarshal(readExample(t, "agent-runtime-request.json"), &agentRequest); err != nil {
		t.Fatalf("unmarshal AgentRequest: %v", err)
	}
	if agentRequest.TaskType == nil || *agentRequest.TaskType != AgentTaskTypeDemo {
		t.Fatalf("agent request task type = %#v", agentRequest.TaskType)
	}
	if agentRequest.ExecutionLimits == nil || agentRequest.ExecutionLimits.MaxSteps != 4 {
		t.Fatalf("agent request limits = %#v", agentRequest.ExecutionLimits)
	}

	var agentResponse AgentResponse
	if err := json.Unmarshal(readExample(t, "agent-runtime-response.json"), &agentResponse); err != nil {
		t.Fatalf("unmarshal AgentResponse: %v", err)
	}
	if agentResponse.Status == nil ||
		*agentResponse.Status != AgentExecutionStatusSucceeded ||
		len(agentResponse.ToolCalls) != 1 {
		t.Fatalf("agent response = %#v", agentResponse)
	}
}

func TestNullablePhaseDeadline(t *testing.T) {
	var event PhaseChanged
	if err := json.Unmarshal(readExample(t, "phase-changed.json"), &event); err != nil {
		t.Fatalf("unmarshal PhaseChanged: %v", err)
	}
	if event.DeadlineAt != nil {
		t.Fatal("DeadlineAt != nil, want null")
	}
}

func TestEnumCompatibility(t *testing.T) {
	events := map[GameEventType]struct{}{
		GameEventTypeGameCreated:         {},
		GameEventTypeRoomPlayerJoined:    {},
		GameEventTypeRoomPlayerLeft:      {},
		GameEventTypeGameStarted:         {},
		GameEventTypeGamePhaseChanged:    {},
		GameEventTypeGameClueUnlocked:    {},
		GameEventTypeGameMessageReceived: {},
		GameEventTypeGameVoteSubmitted:   {},
		GameEventTypeGamePaused:          {},
		GameEventTypeGameResumed:         {},
		GameEventTypeGameEnded:           {},
	}
	if len(events) != 11 {
		t.Fatalf("event enum count = %d, want 11", len(events))
	}

	if ActorTypeDm != "dm" || VisibilityScopePlayer != "player" {
		t.Fatal("generated enum value mismatch")
	}
}

func TestNamedIDTypes(t *testing.T) {
	var userID UserID = "01J9Z3K4M7Q2R8T5V6W1X10000"
	var roomID RoomID = "01J9Z3K4M7Q2R8T5V6W1X20000"

	if string(userID) == string(roomID) {
		t.Fatal("fixture IDs unexpectedly match")
	}
}
