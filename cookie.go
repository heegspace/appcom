package appcom

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// token扩展信息
type TokenExtra struct {
	VipExpire 	int64
}

func (obj TokenExtra) Encode() string {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(obj)
	if nil != err {
		return ""
	}

	data := buf.Bytes()
	return string(data)
}

// 用于设置cookie的结构
type TokenInfo struct {
	UID     int64
	Time    int64
	Token   string
	Role    int64
	Vip     int64
	Expire  int64
	Platom  int64
	Appid   string // appid
	Openid  string // 开发平台id
	Unionid string // 用户唯一id
	Refresh string // token刷新的acess
	Extra   string // 扩展信息
}

func (token TokenInfo) String() string {
	str := "UID: %s Time: %d Token: %s 	Vip: %d  Role: %d  Expire: %d  Platom: %d  Applid: %s  Openid: %s  Unionid: %s  Refresh: %s  Extra: %s"

	return fmt.Sprintf(str, strconv.FormatInt(token.UID, 10), token.Time, token.Token, token.Vip, token.Role, token.Expire, token.Platom, token.Appid, token.Openid, token.Unionid, token.Refresh, token.Extra)
}

func (token TokenInfo) ToExtra() (TokenExtra,error) {
	var extra TokenExtra
	buf := bytes.NewBufferString(token.Extra)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&extra)
	if nil != err {
		return extra,err
	}

	return extra,nil
}

var ivspec = []byte("0000000000000000")

func pkCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)

	return append(ciphertext, padtext...)
}
func pkCS5Trimming(encrypt []byte) []byte {
	padding := encrypt[len(encrypt)-1]

	return encrypt[:len(encrypt)-int(padding)]
}

func aesEncode(src, key string) (value string, err error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		fmt.Println("key error1", err)
		return
	}
	if src == "" {
		fmt.Println("plain content empty")
		err = errors.New("plain content empty")
		return
	}
	ecb := cipher.NewCBCEncrypter(block, ivspec)
	content := []byte(src)
	content = pkCS5Padding(content, block.BlockSize())
	crypted := make([]byte, len(content))
	ecb.CryptBlocks(crypted, content)

	value = hex.EncodeToString(crypted)

	return
}

func aesDecode(token, key string) (subject string, err error) {
	crypted, err := hex.DecodeString(strings.ToLower(token))
	if err != nil || len(crypted) == 0 {
		fmt.Println("plain content empty")
		err = errors.New("plain content empty")

		return
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		fmt.Println("key error1", err)
		err = errors.New("key error1")

		return
	}

	ecb := cipher.NewCBCDecrypter(block, ivspec)
	decrypted := make([]byte, len(crypted))
	ecb.CryptBlocks(decrypted, crypted)

	subject = string(pkCS5Trimming(decrypted))

	return
}

// 使用密钥key对Token数据进行加密
//
// @param src	编码的结构数据
// @param key 	编码的秘钥
//
// @return string,error
//
func EnCookie(src TokenInfo, key string) (token string, err error) {
	if 0 == src.UID {
		err = errors.New("Uid is 0")
		return
	}

	if 0 == len(src.Token) {
		err = errors.New("Token src is empty.")
		return
	}

	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err = enc.Encode(src)
	if nil != err {
		fmt.Println("wirte err ", err)
		return
	}

	data := string(buf.Bytes())
	data = strings.Replace(data, "0", "/", -1)
	data = strings.Replace(data, "1", "!", -1)
	data = strings.Replace(data, "2", "@", -1)
	data = strings.Replace(data, "3", "#", -1)
	data = strings.Replace(data, "7", "$", -1)
	data = strings.Replace(data, "8", "%", -1)
	data = strings.Replace(data, "9", "^", -1)
	data = strings.Replace(data, "f", "++", -1)
	data = strings.Replace(data, "a", "--", -1)
	token, err = aesEncode(data, key)
	if nil != err {
		return
	}
	
	return
}

// 使用秘钥key从src中解码cookie
//
// @param src 	解码数据
// @param key 	秘钥key
// @return TokenInfo,error
//
func DeCookie(src string, key string) (token TokenInfo, err error) {
	if 0 == len(src) {
		err = errors.New("Token info format error.")
		return
	}

	
	obj, err := aesDecode(src, key)
	if nil != err {
		return
	}
	obj = strings.Replace(obj, "/", "0", -1)
	obj = strings.Replace(obj, "!", "1", -1)
	obj = strings.Replace(obj, "@", "2", -1)
	obj = strings.Replace(obj, "#", "3", -1)
	obj = strings.Replace(obj, "$", "7", -1)
	obj = strings.Replace(obj, "%", "8", -1)
	obj = strings.Replace(obj, "^", "9", -1)
	obj = strings.Replace(obj, "++", "f", -1)
	obj = strings.Replace(obj, "--", "a", -1)

	buf := bytes.NewBufferString(obj)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&token)
	if nil != err {
		return
	}

	return
}
