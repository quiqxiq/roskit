package service_test

import (
	"strings"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ParseOnLoginPut ---

func TestParseOnLoginPut_ValidNtf(t *testing.T) {
	putStr := `:put (",ntf,5000,30d,6000,,Disable,Disable,")`
	meta := service.ParseOnLoginPut(putStr)
	require.NotNil(t, meta)
	assert.Equal(t, "ntf", meta.ExpMode)
	assert.Equal(t, "5000", meta.Price)
	assert.Equal(t, "30d", meta.Validity)
	assert.Equal(t, "6000", meta.SellingPrice)
	assert.False(t, meta.NoExpiry)
}

func TestParseOnLoginPut_NoExpiry(t *testing.T) {
	putStr := `:put (",,5000,,6000,noexp,Disable,Disable,")`
	meta := service.ParseOnLoginPut(putStr)
	require.NotNil(t, meta)
	assert.True(t, meta.NoExpiry)
	assert.Equal(t, "0", meta.ExpMode)
}

func TestParseOnLoginPut_NoColonPut(t *testing.T) {
	meta := service.ParseOnLoginPut("not a put string")
	assert.Nil(t, meta)
}

func TestParseOnLoginPut_Empty(t *testing.T) {
	meta := service.ParseOnLoginPut("")
	assert.Nil(t, meta)
}

// --- ParseUserComment ---

func TestParseUserComment_Empty(t *testing.T) {
	expiry, voucher, text := service.ParseUserComment("")
	assert.Empty(t, expiry)
	assert.Empty(t, voucher)
	assert.Empty(t, text)
}

func TestParseUserComment_VCPrefix(t *testing.T) {
	_, voucher, _ := service.ParseUserComment("vc-ABC123")
	assert.Equal(t, "ABC123", voucher)
}

func TestParseUserComment_VCPrefixWithText(t *testing.T) {
	_, voucher, text := service.ParseUserComment("vc-ABC123-mycomment")
	assert.Equal(t, "ABC123", voucher)
	assert.Equal(t, "mycomment", text)
}

func TestParseUserComment_UPPrefix(t *testing.T) {
	_, voucher, _ := service.ParseUserComment("up-XYZ999")
	assert.Equal(t, "XYZ999", voucher)
}

func TestParseUserComment_ExpiryOnly(t *testing.T) {
	// 21-char expiry: "jan/02/2006 15:04:05X" — format checked by position
	comment := "jan/02/2006 15:04:05X"
	expiry, voucher, text := service.ParseUserComment(comment)
	// if pos3=/ pos6=/ pos17=: → it's an expiry
	if comment[3] == '/' && comment[6] == '/' && comment[17] == ':' {
		assert.Equal(t, comment, expiry)
		assert.Empty(t, voucher)
		assert.Empty(t, text)
	} else {
		// falls to plain text path
		assert.Empty(t, expiry)
		assert.Equal(t, comment, text)
	}
}

func TestParseUserComment_PlainText(t *testing.T) {
	_, _, text := service.ParseUserComment("some plain comment")
	assert.Equal(t, "some plain comment", text)
}

// --- GenerateVoucherComment ---

func TestGenerateVoucherComment_WithDate(t *testing.T) {
	result := service.GenerateVoucherComment("vc", "ABC123", "2024-03-15", "note")
	assert.Equal(t, "vc-ABC123-2024-03-15-note", result)
}

func TestGenerateVoucherComment_WithCommentOnly(t *testing.T) {
	result := service.GenerateVoucherComment("vc", "ABC123", "", "mynote")
	assert.Equal(t, "vc-ABC123-mynote", result)
}

func TestGenerateVoucherComment_Minimal(t *testing.T) {
	result := service.GenerateVoucherComment("up", "XYZ999", "", "")
	assert.Equal(t, "up-XYZ999", result)
}

// --- GenerateOnLoginScript ---

func TestGenerateOnLoginScript_EmptyParams(t *testing.T) {
	params := service.OnLoginParams{}
	result := service.GenerateOnLoginScript(params)
	assert.Empty(t, result, "empty params with no expmode or price should return empty string")
}

func TestGenerateOnLoginScript_NoExpiry(t *testing.T) {
	params := service.OnLoginParams{
		ExpMode: "0",
		Price:   "5000",
	}
	result := service.GenerateOnLoginScript(params)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "noexp")
}

func TestGenerateOnLoginScript_NtfMode(t *testing.T) {
	params := service.OnLoginParams{
		ExpMode:  "ntf",
		Price:    "5000",
		Validity: "30d",
	}
	result := service.GenerateOnLoginScript(params)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "API_URL/events/on-login")
	assert.True(t, strings.Contains(result, "ntf") || strings.Contains(result, `"N"`))
}

func TestGenerateOnLoginScript_RemMode(t *testing.T) {
	params := service.OnLoginParams{
		ExpMode:  "rem",
		Price:    "3000",
		Validity: "7d",
	}
	result := service.GenerateOnLoginScript(params)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, `"X"`)
}

func TestGenerateOnLoginScript_UnknownMode(t *testing.T) {
	params := service.OnLoginParams{
		ExpMode: "unknown",
		Price:   "1000",
	}
	result := service.GenerateOnLoginScript(params)
	assert.Empty(t, result, "unknown expmode should return empty string")
}
