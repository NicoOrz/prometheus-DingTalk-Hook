package dingtalk

import (
	"net/url"
	"testing"
)

func TestSign_KnownValue(t *testing.T) {
	got := Sign(1700000000000, "secret")
	const want = "OuzzJR5+xZ4/EYwqtNt6sMYZQMTa/HEGvc9miJe7XzY="
	if got != want {
		t.Fatalf("Sign() = %q, want %q", got, want)
	}
}

func TestSign_URLRoundTrip(t *testing.T) {
	u, err := url.Parse("https://oapi.dingtalk.com/robot/send?access_token=xxx")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	q := u.Query()
	q.Set("timestamp", "1700000000000")
	q.Set("sign", Sign(1700000000000, "secret"))
	u.RawQuery = q.Encode()

	parsed, err := url.Parse(u.String())
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	if parsed.Query().Get("sign") != Sign(1700000000000, "secret") {
		t.Fatalf("sign did not round-trip")
	}
}
