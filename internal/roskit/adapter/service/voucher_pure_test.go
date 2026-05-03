package service_test

import (
	"strings"
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateVoucherCode_Length(t *testing.T) {
	for _, length := range []int{4, 8, 12, 16} {
		code := service.GenerateVoucherCode("mix", length)
		assert.Len(t, code, length)
	}
}

func TestGenerateVoucherCode_Charsets(t *testing.T) {
	lower := "abcdefghijklmnopqrstuvwxyz"
	upper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"

	tests := []struct {
		charset  string
		allowed  string
	}{
		{"lower", lower},
		{"upper", upper},
		{"num", digits},
		{"upplow", upper + lower},
	}
	for _, tt := range tests {
		code := service.GenerateVoucherCode(tt.charset, 20)
		for _, c := range code {
			assert.True(t, strings.ContainsRune(tt.allowed, c),
				"charset %q: unexpected char %q in %q", tt.charset, c, code)
		}
	}
}

func TestGeneratePassword_OnlyDigits(t *testing.T) {
	for i := 0; i < 10; i++ {
		pass := service.GeneratePassword(8)
		assert.Len(t, pass, 8)
		for _, c := range pass {
			assert.True(t, c >= '0' && c <= '9',
				"expected digit, got %q", c)
		}
	}
}

func TestGenerateVoucher_TypeVC(t *testing.T) {
	params := service.VoucherParams{
		Prefix:     "vc",
		CharSet:    "lower",
		NameLength: 6,
		UserType:   "vc",
	}
	v := service.GenerateVoucher(params)
	assert.True(t, strings.HasPrefix(v.Username, "vc"))
	assert.Equal(t, v.Username, v.Password, "vc type: password should equal username")
}

func TestGenerateVoucher_TypeUP(t *testing.T) {
	params := service.VoucherParams{
		Prefix:     "",
		CharSet:    "mix",
		NameLength: 8,
		UserType:   "up",
	}
	v := service.GenerateVoucher(params)
	assert.Len(t, v.Password, 8)
	for _, c := range v.Password {
		assert.True(t, c >= '0' && c <= '9', "up type: password must be digits only")
	}
}

func TestGenerateVoucherBatch_Count(t *testing.T) {
	params := service.VoucherParams{
		Qty:        5,
		CharSet:    "mix",
		NameLength: 8,
		UserType:   "vc",
	}
	vouchers := service.GenerateVoucherBatch(params)
	assert.Len(t, vouchers, 5)
}

func TestGenerateVoucherBatch_Unique(t *testing.T) {
	params := service.VoucherParams{
		Qty:        20,
		CharSet:    "mix2",
		NameLength: 8,
		UserType:   "vc",
	}
	vouchers := service.GenerateVoucherBatch(params)
	require.Len(t, vouchers, 20)

	seen := make(map[string]bool)
	for _, v := range vouchers {
		assert.False(t, seen[v.Username], "duplicate username: %s", v.Username)
		seen[v.Username] = true
	}
}
