package mikrotik

import (
	"testing"
	"time"
)

func TestIsDead(t *testing.T) {
	tests := []struct {
		name   string
		pairs  map[string]string
		expect bool
	}{
		{"dead=true", map[string]string{".dead": "true"}, true},
		{"dead=yes", map[string]string{".dead": "yes"}, true},
		{"dead=false", map[string]string{".dead": "false"}, false},
		{"no dead key", map[string]string{"name": "ether1"}, false},
		{"empty map", map[string]string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDead(tt.pairs)
			if got != tt.expect {
				t.Errorf("IsDead(%v) = %v, want %v", tt.pairs, got, tt.expect)
			}
		})
	}
}

func TestParseUint64(t *testing.T) {
	tests := []struct {
		input  string
		expect uint64
	}{
		{"0", 0},
		{"12345", 12345},
		{"18446744073709551615", 18446744073709551615},
		{"", 0},
		{"abc", 0},
		{"-1", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseUint64(tt.input)
			if got != tt.expect {
				t.Errorf("ParseUint64(%q) = %d, want %d", tt.input, got, tt.expect)
			}
		})
	}
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		input  string
		expect int64
	}{
		{"0", 0},
		{"12345", 12345},
		{"-99", -99},
		{"", 0},
		{"abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseInt64(tt.input)
			if got != tt.expect {
				t.Errorf("ParseInt64(%q) = %d, want %d", tt.input, got, tt.expect)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input  string
		expect bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"yes", true},
		{"Yes", true},
		{"false", false},
		{"no", false},
		{"", false},
		{"abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseBool(tt.input)
			if got != tt.expect {
				t.Errorf("ParseBool(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input  string
		expect time.Duration
	}{
		{"1w2d3h4m5s", 1*7*24*time.Hour + 2*24*time.Hour + 3*time.Hour + 4*time.Minute + 5*time.Second},
		{"3h25m", 3*time.Hour + 25*time.Minute},
		{"45s", 45 * time.Second},
		{"1d", 24 * time.Hour},
		{"2w", 2 * 7 * 24 * time.Hour},
		{"1h30m15s", 1*time.Hour + 30*time.Minute + 15*time.Second},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseDuration(tt.input)
			if got != tt.expect {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestSentenceToMap(t *testing.T) {
	words := []string{
		"=name=ether1",
		"=type=ether",
		"=rx-byte=1024",
		"=.id=*1",
	}

	m := SentenceToMap(words)

	expected := map[string]string{
		"name":    "ether1",
		"type":    "ether",
		"rx-byte": "1024",
		".id":     "*1",
	}

	for k, v := range expected {
		if m[k] != v {
			t.Errorf("SentenceToMap[%q] = %q, want %q", k, m[k], v)
		}
	}
}

func TestFormatCacheKey(t *testing.T) {
	key := FormatCacheKey("core-01", "interface_stats", "ether1")
	expected := "roskit:core-01:interface_stats:ether1"
	if key != expected {
		t.Errorf("FormatCacheKey = %q, want %q", key, expected)
	}
}

func TestFormatPubSubChannel(t *testing.T) {
	ch := FormatPubSubChannel("core-01")
	expected := "roskit:telemetry:core-01"
	if ch != expected {
		t.Errorf("FormatPubSubChannel = %q, want %q", ch, expected)
	}
}
