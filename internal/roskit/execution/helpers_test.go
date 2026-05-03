package execution

import (
	"errors"
	"testing"

	routeros "github.com/go-routeros/routeros/v3"
	"github.com/stretchr/testify/assert"
)

// --- ConnState.String ---

func TestConnState_String(t *testing.T) {
	tests := []struct {
		state    ConnState
		expected string
	}{
		{ConnStateDisconnected, "disconnected"},
		{ConnStateConnecting, "connecting"},
		{ConnStateConnected, "connected"},
		{ConnStateAuthFailed, "auth_failed"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.state.String())
	}
}

// --- isAuthError ---

func TestIsAuthError(t *testing.T) {
	assert.True(t, isAuthError(errors.New("cannot log in")))
	assert.True(t, isAuthError(errors.New("CANNOT LOG IN")))
	assert.True(t, isAuthError(errors.New("invalid user name or password")))
	assert.True(t, isAuthError(errors.New("authentication failed")))
	assert.True(t, isAuthError(errors.New("dial: cannot log in to router")))

	assert.False(t, isAuthError(nil))
	assert.False(t, isAuthError(errors.New("connection refused")))
	assert.False(t, isAuthError(errors.New("i/o timeout")))
	assert.False(t, isAuthError(errors.New("EOF")))
}

// --- ConnConfig.withDefaults ---

func TestConnConfig_WithDefaults_ZeroValues(t *testing.T) {
	cfg := ConnConfig{RouterID: "r1", Address: "192.168.1.1:8728"}
	result := cfg.withDefaults()

	assert.Equal(t, 10_000_000_000, int(result.DialTimeout))    // 10s
	assert.Equal(t, 15_000_000_000, int(result.ReadTimeout))    // 15s
	assert.Equal(t, 15_000_000_000, int(result.WriteTimeout))   // 15s
	assert.Equal(t, 30_000_000_000, int(result.HealthInterval)) // 30s
}

func TestConnConfig_WithDefaults_PreservesExisting(t *testing.T) {
	cfg := ConnConfig{
		RouterID:       "r1",
		DialTimeout:    5_000_000_000,
		HealthInterval: 60_000_000_000,
	}
	result := cfg.withDefaults()
	assert.Equal(t, 5_000_000_000, int(result.DialTimeout))
	assert.Equal(t, 60_000_000_000, int(result.HealthInterval))
	assert.Equal(t, 15_000_000_000, int(result.ReadTimeout))
}

// --- buildAdd / buildSet / buildRemove ---

func TestBuildAdd(t *testing.T) {
	sentence := buildAdd("ip/hotspot/user", map[string]string{
		"name":     "user001",
		"password": "pass001",
	})
	assert.Equal(t, "/ip/hotspot/user/add", sentence[0])
	assert.Contains(t, sentence, "=name=user001")
	assert.Contains(t, sentence, "=password=pass001")
}

func TestBuildSet(t *testing.T) {
	sentence := buildSet("ip/hotspot/user", "*1", map[string]string{
		"disabled": "true",
	})
	assert.Equal(t, "/ip/hotspot/user/set", sentence[0])
	assert.Equal(t, "=.id=*1", sentence[1])
	assert.Contains(t, sentence, "=disabled=true")
}

func TestBuildRemove(t *testing.T) {
	sentence := buildRemove("ip/hotspot/user", "*1")
	assert.Equal(t, []string{"/ip/hotspot/user/remove", "=.id=*1"}, sentence)
}

func TestBuildAdd_EmptyParams(t *testing.T) {
	sentence := buildAdd("ip/hotspot/user", map[string]string{})
	assert.Equal(t, []string{"/ip/hotspot/user/add"}, sentence)
}

// --- Executor.ExtractID ---

func TestExtractID_NilReply(t *testing.T) {
	e := &Executor{}
	// nil Done → empty string
	assert.Equal(t, "", e.ExtractID(&routeros.Reply{}))
}

func TestExtractID_NilDone(t *testing.T) {
	e := &Executor{}
	assert.Equal(t, "", e.ExtractID(&routeros.Reply{Done: nil}))
}
