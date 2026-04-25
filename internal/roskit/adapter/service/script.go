package service

import (
	"fmt"
	"strings"
)

type OnLoginParams struct {
	ExpMode      string
	Price        string
	SellingPrice string
	Validity     string
	ProfileName  string
	LockUser     string
	LockServer   string
}

type OnLoginMetadata struct {
	ExpMode      string
	Price        string
	Validity     string
	SellingPrice string
	NoExpiry     bool
	LockUser     string
	LockServer   string
}

func GenerateOnLoginScript(params OnLoginParams) string {
	expmode := params.ExpMode
	price := params.Price
	if price == "" {
		price = "0"
	}
	sprice := params.SellingPrice
	if sprice == "" {
		sprice = "0"
	}
	validity := strings.ToLower(params.Validity)
	name := params.ProfileName
	lockuser := params.LockUser
	lockserver := params.LockServer

	if expmode == "" && price == "0" {
		return ""
	}

	lock := ""
	if lockuser == "Enable" {
		lock = `; [:local mac $"mac-address"; /ip hotspot user set mac-address=$mac [find where name=$user]]`
	}

	slock := ""
	if lockserver != "Disable" && lockserver != "" {
		slock = `; [:local mac $"mac-address"; :local srv [/ip hotspot host get [find where mac-address="$mac"] server]; /ip hotspot user set server=$srv [find where name=$user]]`
	}

	if expmode == "0" {
		return fmt.Sprintf(`:put (",,%s,,%s,noexp,%s,%s,")%s%s`,
			price, sprice, lockuser, lockserver, lock, slock)
	}

	var mode string
	var hasRecord bool
	switch expmode {
	case "ntf":
		mode = "N"
		hasRecord = false
	case "ntfc":
		mode = "N"
		hasRecord = true
	case "rem":
		mode = "X"
		hasRecord = false
	case "remc":
		mode = "X"
		hasRecord = true
	default:
		return ""
	}

	put := fmt.Sprintf(`:put (",%s,%s,%s,%s,,%s,%s,")`,
		expmode, price, validity, sprice, lockuser, lockserver)

	fetch := ""
	if expmode != "" || price != "0" {
		fetch = `/tool/fetch mode=http url="API_URL/events/on-login" http-data="router_session=SESSION&server=$server&username=$user&mac=$mac-address&ip=$address&date=$date&time=$time&profile=[/ip hotspot user get [find where name=$user] profile]" as-value output=no; `
	}

	record := ""
	if hasRecord {
		record = fmt.Sprintf(
			`; :local mac $"mac-address"; :local time [/system clock get time ]; `+
				`/system script add name="$date-|-$time-|-$user-|-%s-|-$address-|-$mac-|-%s-|-%s-|-$comment" `+
				`owner="$month$year" source=$date comment=mikhmon`,
			price, validity, name,
		)
	}

	expiry := fmt.Sprintf(
		`:local mode "%s"; `+
			`{:local date [ /system clock get date ]; `+
			`:local year [ :pick $date 7 11 ]; `+
			`:local month [ :pick $date 0 3 ]; `+
			`:local comment [ /ip hotspot user get [/ip hotspot user find where name="$user"] comment]; `+
			`:local ucode [:pic $comment 0 2]; `+
			`:if ($ucode = "vc" or $ucode = "up" or $comment = "") do={`+
			`/sys sch add name="$user" disable=no start-date=$date interval="%s"; `+
			`:delay 2s; `+
			`:local exp [ /sys sch get [ /sys sch find where name="$user" ] next-run]; `+
			`:local getxp [len $exp]; `+
			`:if ($getxp = 15) do={`+
			`:local d [:pic $exp 0 6]; :local t [:pic $exp 7 16]; :local s ("/"); :local exp ("$d$s$year $t"); `+
			`/ip hotspot user set comment="$exp $mode" [find where name=$user]}; `+
			`:if ($getxp = 8) do={`+
			`/ip hotspot user set comment="$date $exp $mode" [find where name=$user]}; `+
			`:if ($getxp > 15) do={`+
			`/ip hotspot user set comment="$exp $mode" [find where name=$user]}; `+
			`/sys sch remove [find where name="$user"]}`,
		mode, validity,
	)

	script := fmt.Sprintf(`%s %s %s %s}%s%s`, fetch, put, expiry, record, lock, slock)
	script = strings.ReplaceAll(script, "\n", " ")
	return script
}

func ParseOnLoginPut(putStr string) *OnLoginMetadata {
	putStr = strings.TrimSpace(putStr)
	if !strings.Contains(putStr, ":put") {
		return nil
	}

	start := strings.Index(putStr, "(")
	end := strings.Index(putStr, ")")
	if start < 0 || end < 0 || end <= start {
		return nil
	}

	content := putStr[start+1 : end]
	parts := strings.Split(content, ",")

	meta := &OnLoginMetadata{}
	if len(parts) > 1 {
		meta.ExpMode = parts[1]
	}
	if len(parts) > 2 {
		meta.Price = parts[2]
	}
	if len(parts) > 3 {
		meta.Validity = parts[3]
	}
	if len(parts) > 4 {
		meta.SellingPrice = parts[4]
	}
	if len(parts) > 5 && parts[5] == "noexp" {
		meta.NoExpiry = true
		meta.ExpMode = "0"
	}
	if len(parts) > 6 {
		meta.LockUser = parts[6]
	}
	if len(parts) > 7 {
		meta.LockServer = parts[7]
	}

	return meta
}

func ParseUserComment(comment string) (expiry string, voucherCode string, text string) {
	if len(comment) == 0 {
		return "", "", ""
	}

	prefix := comment
	if len(comment) > 2 {
		prefix = comment[:2]
	}

	if prefix == "vc" || prefix == "up" {
		parts := strings.SplitN(comment, "-", 4)
		switch len(parts) {
		case 2:
			return "", parts[1], ""
		case 3:
			return "", parts[1], parts[2]
		case 4:
			return "", parts[1] + "-" + parts[2], parts[3]
		}
	}

	if len(comment) == 21 {
		pos3 := comment[3:4]
		pos6 := comment[6:7]
		pos17 := comment[17:18]
		if pos3 == "/" && pos6 == "/" && pos17 == ":" {
			return comment, "", ""
		}
	}

	if len(comment) > 21 {
		pos3 := comment[3:4]
		pos6 := comment[6:7]
		pos17 := comment[17:18]
		if pos3 == "/" && pos6 == "/" && pos17 == ":" {
			expPart := comment[:21]
			return expPart, "", comment[22:]
		}
	}

	return "", "", comment
}

func GenerateVoucherComment(userType, gencode, date, gcomment string) string {
	if date != "" {
		return fmt.Sprintf("%s-%s-%s-%s", userType, gencode, date, gcomment)
	}
	if gcomment != "" {
		return fmt.Sprintf("%s-%s-%s", userType, gencode, gcomment)
	}
	return fmt.Sprintf("%s-%s", userType, gencode)
}
