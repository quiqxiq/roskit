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
