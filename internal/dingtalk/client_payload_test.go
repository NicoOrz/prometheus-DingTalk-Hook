package dingtalk

import (
	"encoding/json"
	"testing"
)

func TestBuildPayload_MarkdownAt(t *testing.T) {
	b, err := buildPayload(Message{
		MsgType:  "markdown",
		Title:    "t",
		Markdown: "hello",
		At: &At{
			AtMobiles: []string{"13800138000"},
			AtUserIds: []string{"user123"},
			IsAtAll:   true,
		},
	})
	if err != nil {
		t.Fatalf("buildPayload: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	at, ok := payload["at"].(map[string]any)
	if !ok {
		t.Fatalf("missing at field: %v", payload)
	}
	if at["isAtAll"] != true {
		t.Fatalf("isAtAll=%v", at["isAtAll"])
	}

	mobiles, ok := at["atMobiles"].([]any)
	if !ok || len(mobiles) != 1 || mobiles[0] != "13800138000" {
		t.Fatalf("atMobiles=%v", at["atMobiles"])
	}

	userIds, ok := at["atUserIds"].([]any)
	if !ok || len(userIds) != 1 || userIds[0] != "user123" {
		t.Fatalf("atUserIds=%v", at["atUserIds"])
	}

	md, ok := payload["markdown"].(map[string]any)
	if !ok {
		t.Fatalf("missing markdown field: %v", payload)
	}
	if md["text"] != "hello\n\n@all @user123 @13800138000" {
		t.Fatalf("markdown.text=%q", md["text"])
	}
}

func TestBuildPayload_EmptyAtOmitted(t *testing.T) {
	b, err := buildPayload(Message{
		MsgType:  "text",
		Text:     "hello",
		At:       &At{},
		Markdown: "",
	})
	if err != nil {
		t.Fatalf("buildPayload: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, ok := payload["at"]; ok {
		t.Fatalf("unexpected at field: %v", payload["at"])
	}
}
