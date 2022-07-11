package appcom

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CookieInfo struct {
	Jyauth string `json:"hgauth" form:"hgauth"`
	Token  string `json:"__RequestVerificationToken" form:"__RequestVerificationToken"`
}

// 解析cookie数据
// 将头部的cookie或url中的cookie数据解析出来
// 有限解析head中的数据
//
func parseCookie(c *gin.Context) (auth CookieInfo, err error) {
	var cookie CookieInfo
	cookie.Jyauth, err = c.Cookie("hgauth")
	if nil != err {
		goto query
	}

	cookie.Token, err = c.Cookie("__RequestVerificationToken")
	if nil != err {
		goto query
	}

	goto token

query:
	err = c.ShouldBindQuery(&cookie)
	if nil != err {
		c.String(http.StatusUnauthorized, "NOT LOGIN")
		c.Abort()

		return
	}

token:
	if cookie.Token != cookie.Jyauth {
		c.String(http.StatusUnauthorized, "AUTH_ERROR")
		c.Abort()

		return
	}

	auth = cookie
	return
}

// 权限中间件，主要是确认是否登陆成功，设置一个回调函数已在本地服务器中确认
//
// @param callback 	回调函数，用于回传cookie数据
// @param timeout 	token的过期时间
//
func NeedLogin(callback func(c *gin.Context, cookie CookieInfo) bool, timeout int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now().UnixNano()

		for k, v := range c.Request.Header {
			fmt.Println(k, v)
		}

		cookie, err := parseCookie(c)
		if nil != err {
			c.String(http.StatusUnauthorized, "NOT_LOGIN"+err.Error())
			c.Abort()

			return
		}

		if cookie.Token != cookie.Jyauth {
			c.String(http.StatusUnauthorized, "AUTH_ERROR")
			c.Abort()

			return
		}

		// 在对应的服务中验证登录是否有效 //
		if !callback(c, cookie) {
			c.String(http.StatusUnauthorized, "NOT_LOGIN")
			c.Abort()

			return
		}

		end := time.Now().UnixNano()
		fmt.Println("t--->", end-now)

		c.Next()
	}
}

// 权限中间件，主要是获取cookie中的信息
//
// @param callback 	回调函数，用于回传cookie数据
// @param timeout 	token的过期时间
//
func NeedCookie(callback func(c *gin.Context, cookie CookieInfo) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		for k, v := range c.Request.Header {
			fmt.Println(k, v)
		}

		var ck CookieInfo
		jyauth, err := c.Cookie("hgauth")
		if nil != err {
			c.Next()

			return
		}
		ck.Jyauth = jyauth

		token, err := c.Cookie("__RequestVerificationToken")
		if nil != err {
			c.Next()

			return
		}
		ck.Token = token

		callback(c, ck)
		c.Next()
	}
}

// 权限中间件，检测对应的访问是否收到限制
//
// @param cb 	回调函数，用于回传cookie数据
//	 返回：
//		int		策略状态
//		string	策略信息
// @param	openFn 	是否打开了限制验证
//
func IsLimited(cb func(c *gin.Context, cookie CookieInfo, uniqueid string) (bool, string), openFn func(c *gin.Context) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		for k, v := range c.Request.Header {
			fmt.Println(k, v)
		}

		if !openFn(c) {
			c.Next()

			return
		}

		// 获取唯一id
		uniqueId := c.Request.Header.Get("Tag-Unid")
		myId := c.Request.Header.Get("My-Id")
		Bidden := c.Request.Header.Get("B-Idden")
		if (0 == len(uniqueId) && 0 == len(myId) && 0 == len(Bidden)) || 0 == len(Bidden) {
			// 403 禁止访问
			c.String(http.StatusForbidden, "Forbidden, 请你通过正规方式访问！")
			c.Abort()

			return
		}
		bids := strings.Split(Bidden, "/")
		if 2 != len(bids) {
			// 403 禁止访问
			c.String(http.StatusForbidden, "请你合理访问，否则将追究法律责任")
			c.Abort()

			return
		}
		Bidden = bids[0]
		sign := bids[1]
		sigs := strings.Split(sign, "-")
		if 2 != len(sigs) {
			// 403 禁止访问
			c.String(http.StatusForbidden, "你的行为已被监控，请合理访问，否则将追究法律责任")
			c.Abort()

			return
		}
		// 时间戳
		// stamp := sigs[1]

		var ck CookieInfo
		jyauth, err := c.Cookie("hgauth")
		if nil == err {
			ck.Jyauth = jyauth
		}
		token, err := c.Cookie("__RequestVerificationToken")
		if nil == err {
			ck.Token = token
		}

		// 405 访问受限
		is, strategy := cb(c, ck, Bidden)
		if is {
			c.String(http.StatusMethodNotAllowed, strategy)
			c.Abort()

			return
		}

		c.Set("uniqueid", uniqueId)
		c.Next()
	}
}
