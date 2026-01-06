package config

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestDuration_JSON(t *testing.T) {
	t.Parallel()

	b, err := json.Marshal(Duration(5 * time.Second))
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(b) != `"5s"` {
		t.Fatalf("json=%s", b)
	}

	var got Duration
	if err := json.Unmarshal([]byte(`"10s"`), &got); err != nil {
		t.Fatalf("json.Unmarshal string: %v", err)
	}
	if got.Duration() != 10*time.Second {
		t.Fatalf("got=%s", got.Duration())
	}

	if err := json.Unmarshal([]byte(`10`), &got); err != nil {
		t.Fatalf("json.Unmarshal number: %v", err)
	}
	if got.Duration() != 10*time.Second {
		t.Fatalf("got=%s", got.Duration())
	}

	if err := json.Unmarshal([]byte(`"10"`), &got); err != nil {
		t.Fatalf("json.Unmarshal numeric string: %v", err)
	}
	if got.Duration() != 10*time.Second {
		t.Fatalf("got=%s", got.Duration())
	}
}

func TestDuration_YAMLMarshal_String(t *testing.T) {
	t.Parallel()

	type wrapper struct {
		D Duration `yaml:"d"`
	}

	b, err := yaml.Marshal(wrapper{D: Duration(5 * time.Second)})
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	if !strings.Contains(string(b), "d: 5s") {
		t.Fatalf("yaml=%q", string(b))
	}
}
