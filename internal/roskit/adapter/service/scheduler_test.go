package service_test

import (
	"strings"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/stretchr/testify/assert"
)

func TestGenerateExpireMonitorScript_NotEmpty(t *testing.T) {
	script := service.GenerateExpireMonitorScript()
	assert.NotEmpty(t, script)
}

func TestGenerateExpireMonitorScript_ContainsKeyLogic(t *testing.T) {
	script := service.GenerateExpireMonitorScript()
	assert.True(t, strings.Contains(script, "hotspot user"), "should contain 'hotspot user'")
	assert.True(t, strings.Contains(script, "dateint"), "should contain 'dateint'")
	assert.True(t, strings.Contains(script, "timeint"), "should contain 'timeint'")
	assert.True(t, strings.Contains(script, "limit-uptime"), "should contain 'limit-uptime'")
}

func TestExpireMonitorName(t *testing.T) {
	assert.Equal(t, "Mikhmon-Expire-Monitor", service.ExpireMonitorName)
}

func TestExpireMonitorComment_VersionTag(t *testing.T) {
	assert.Equal(t, "Mikhmon Expire Monitor v2 [roskit]", service.ExpireMonitorComment)
	assert.True(t, strings.Contains(service.ExpireMonitorComment, "v2"),
		"comment must include a version tag so future deploys can detect-and-upgrade legacy installs")
	assert.True(t, strings.Contains(service.ExpireMonitorComment, "roskit"),
		"comment must distinguish the roskit-managed scheduler from the legacy mikhmon one")
}

func TestGenerateExpireMonitorScript_SemicolonSeparated(t *testing.T) {
	script := service.GenerateExpireMonitorScript()
	assert.False(t, strings.Contains(script, "\n"),
		"script must be flattened to a one-line `;`-separated form for inline on-event embedding")
	assert.True(t, strings.Contains(script, "; "),
		"flattened script must use `; ` between statements")
}
