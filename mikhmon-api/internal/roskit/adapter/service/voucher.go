package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type VoucherParams struct {
	Qty        int
	Server     string
	UserType   string
	NameLength int
	Prefix     string
	CharSet    string
	Profile    string
	TimeLimit  string
	DataLimit  string
	Comment    string
	Gencode    string
}

type GeneratedVoucher struct {
	Username string
	Password string
}

const (
	lowerLetters  = "abcdefghijklmnopqrstuvwxyz"
	upperLetters  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits        = "0123456789"
	allAlpha      = lowerLetters + upperLetters
	alphaNum      = lowerLetters + digits
	alphaNumUpper = upperLetters + digits
	allAlphaNum   = lowerLetters + upperLetters + digits
)

func GenerateVoucherCode(charset string, length int) string {
	var chars string
	switch charset {
	case "lower":
		chars = lowerLetters
	case "upper":
		chars = upperLetters
	case "upplow":
		chars = allAlpha
	case "mix":
		chars = alphaNum
	case "mix1":
		chars = alphaNumUpper
	case "mix2":
		chars = allAlphaNum
	case "num":
		chars = digits
	case "lower1":
		chars = lowerLetters
	case "upper1":
		chars = upperLetters
	case "upplow1":
		chars = allAlpha
	default:
		chars = alphaNum
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[r.Intn(len(chars))]
	}
	return string(result)
}

func GeneratePassword(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = digits[r.Intn(len(digits))]
	}
	return string(result)
}

func GenerateVoucher(params VoucherParams) GeneratedVoucher {
	username := params.Prefix + GenerateVoucherCode(params.CharSet, params.NameLength)

	var password string
	switch params.UserType {
	case "vc":
		password = username
	case "up":
		password = GeneratePassword(params.NameLength)
	default:
		password = username
	}

	return GeneratedVoucher{
		Username: username,
		Password: password,
	}
}

func GenerateVoucherBatch(params VoucherParams) []GeneratedVoucher {
	vouchers := make([]GeneratedVoucher, 0, params.Qty)
	seen := make(map[string]bool)

	for len(vouchers) < params.Qty {
		v := GenerateVoucher(params)
		if !seen[v.Username] {
			seen[v.Username] = true
			vouchers = append(vouchers, v)
		}
	}

	return vouchers
}

func (b *Bridge) GenerateAndCreateVouchers(ctx context.Context, routerID string, params VoucherParams) ([]GeneratedVoucher, error) {
	vouchers := GenerateVoucherBatch(params)

	batchSize := 50
	for i := 0; i < len(vouchers); i += batchSize {
		end := i + batchSize
		if end > len(vouchers) {
			end = len(vouchers)
		}

		batch := vouchers[i:end]
		for _, v := range batch {
			userParams := map[string]string{
				"name":     v.Username,
				"password": v.Password,
				"profile":  params.Profile,
				"comment":  params.Comment,
				"disabled": "no",
			}
			if params.Server != "" {
				userParams["server"] = params.Server
			}
			if params.TimeLimit != "" {
				userParams["limit-uptime"] = params.TimeLimit
			}
			if params.DataLimit != "" {
				userParams["limit-bytes-total"] = params.DataLimit
			}

			if _, err := b.mutateAdd(ctx, routerID, "ip/hotspot/user/add", userParams); err != nil {
				return vouchers[:i], fmt.Errorf("batch failed at user %s: %w", v.Username, err)
			}
		}
	}

	return vouchers, nil
}
