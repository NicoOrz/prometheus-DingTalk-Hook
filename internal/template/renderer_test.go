package template

import (
	"strings"
	"testing"

	"prometheus-dingtalk-hook/internal/alertmanager"
	"prometheus-dingtalk-hook/internal/config"
)

func TestRender_DefaultTemplate(t *testing.T) {
	r, err := NewRenderer(config.TemplateConfig{})
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	out, err := r.Render("", alertmanager.WebhookMessage{
		Receiver: "default",
		Status:   "firing",
		Alerts: []alertmanager.Alert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighCPU",
				},
				Annotations: map[string]string{
					"summary": "cpu too high",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "Receiver:") || !strings.Contains(out, "default") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "HighCPU") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "告警触发（1）") {
		t.Fatalf("unexpected output: %q", out)
	}
}
