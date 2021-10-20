package appcom

import (
	"fmt"
	"testing"
	"time"
)

func TestEnCookie(t *testing.T) {
	var tokenExtra TokenExtra
	tokenExtra.VipExpire = 999

	var token TokenInfo
	token.UID = 122222111111111111
	token.Time = 123123
	token.Token = "asdf2112341234"
	token.Role = 1
	token.Expire = time.Now().Unix()
	token.Extra = tokenExtra.Encode()

	cookie, err := EnCookie(token, "12345678901234567890123456789098")
	if nil != err {
		fmt.Println(err)
		return
	}

	fmt.Println("cookie: ", cookie)
	obj, err := DeCookie(cookie, "12345678901234567890123456789098")
	if nil != err {
		fmt.Println(err)
		return
	}

	fmt.Println(obj)

	extra,_ := obj.ToExtra()
	fmt.Println(extra)
	return
}
