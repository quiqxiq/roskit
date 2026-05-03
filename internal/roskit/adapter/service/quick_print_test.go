package service_test

import (
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuickPrintSource_Valid(t *testing.T) {
	src := "#TestPkg#all#vc#6#qp#mix#default#1d#0##1d#1000_500#yes"
	pkg := service.ParseQuickPrintSource(src)
	require.NotNil(t, pkg)
	assert.Equal(t, "TestPkg", pkg.Name)
	assert.Equal(t, "all", pkg.Server)
	assert.Equal(t, "vc", pkg.UserMode)
	assert.Equal(t, "6", pkg.UserLength)
	assert.Equal(t, "qp", pkg.Prefix)
	assert.Equal(t, "mix", pkg.CharMode)
	assert.Equal(t, "default", pkg.Profile)
	assert.Equal(t, "1d", pkg.TimeLimit)
	assert.Equal(t, "0", pkg.DataLimit)
	assert.Equal(t, "", pkg.Comment)
	assert.Equal(t, "1d", pkg.Validity)
	assert.Equal(t, "1000", pkg.Price)
	assert.Equal(t, "500", pkg.SellingPrice)
	assert.Equal(t, "yes", pkg.LockUser)
}

func TestParseQuickPrintSource_TooFewParts(t *testing.T) {
	src := "#name#server#vc#6#qp"
	pkg := service.ParseQuickPrintSource(src)
	assert.Nil(t, pkg)
}

func TestParseQuickPrintSource_Empty(t *testing.T) {
	pkg := service.ParseQuickPrintSource("")
	assert.Nil(t, pkg)
}

func TestQuickPrintPackage_ToSource(t *testing.T) {
	pkg := &service.QuickPrintPackage{
		Name:         "TestPkg",
		Server:       "all",
		UserMode:     "vc",
		UserLength:   "6",
		Prefix:       "qp",
		CharMode:     "mix",
		Profile:      "default",
		TimeLimit:    "1d",
		DataLimit:    "0",
		Comment:      "",
		Validity:     "1d",
		Price:        "1000",
		SellingPrice: "500",
		LockUser:     "yes",
	}
	result := pkg.ToSource()
	assert.Equal(t, "#TestPkg#all#vc#6#qp#mix#default#1d#0##1d#1000_500#yes", result)
}

func TestQuickPrintPackage_Roundtrip(t *testing.T) {
	pkg := &service.QuickPrintPackage{
		Name:         "MyPkg",
		Server:       "hotspot1",
		UserMode:     "up",
		UserLength:   "8",
		Prefix:       "pr",
		CharMode:     "num",
		Profile:      "premium",
		TimeLimit:    "3d",
		DataLimit:    "1G",
		Comment:      "test comment",
		Validity:     "3d",
		Price:        "2000",
		SellingPrice: "1500",
		LockUser:     "no",
	}
	src := pkg.ToSource()
	parsed := service.ParseQuickPrintSource(src)
	require.NotNil(t, parsed)
	assert.Equal(t, pkg.Name, parsed.Name)
	assert.Equal(t, pkg.Server, parsed.Server)
	assert.Equal(t, pkg.UserMode, parsed.UserMode)
	assert.Equal(t, pkg.UserLength, parsed.UserLength)
	assert.Equal(t, pkg.Prefix, parsed.Prefix)
	assert.Equal(t, pkg.CharMode, parsed.CharMode)
	assert.Equal(t, pkg.Profile, parsed.Profile)
	assert.Equal(t, pkg.TimeLimit, parsed.TimeLimit)
	assert.Equal(t, pkg.DataLimit, parsed.DataLimit)
	assert.Equal(t, pkg.Comment, parsed.Comment)
	assert.Equal(t, pkg.Validity, parsed.Validity)
	assert.Equal(t, pkg.Price, parsed.Price)
	assert.Equal(t, pkg.SellingPrice, parsed.SellingPrice)
	assert.Equal(t, pkg.LockUser, parsed.LockUser)
}

func TestQuickPrintPackage_PriceSplit(t *testing.T) {
	src := "#Pkg#all#vc#6###default#1d#0##1d#1000_500#yes"
	pkg := service.ParseQuickPrintSource(src)
	require.NotNil(t, pkg)
	assert.Equal(t, "1000", pkg.Price)
	assert.Equal(t, "500", pkg.SellingPrice)
}

func TestQuickPrintPackage_NoSellingPrice(t *testing.T) {
	src := "#Pkg#all#vc#6###default#1d#0##1d#1000#yes"
	pkg := service.ParseQuickPrintSource(src)
	require.NotNil(t, pkg)
	assert.Equal(t, "1000", pkg.Price)
	assert.Equal(t, "", pkg.SellingPrice)
}
