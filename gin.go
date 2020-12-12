package appcom

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// gin框架的Logger中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		if raw != "" {
			path = path + "?" + raw
		}

		fmt.Println(path)

		c.Next()
	}
}

// 获取客户端ip
//
func GetIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if nil != net.ParseIP(ip) {
		return ip
	}

	ip = r.Header.Get("X-Forward-For")
	for _, i := range strings.Split(ip, ",") {
		if nil != net.ParseIP(i) {
			return i
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if nil != err {
		return "127.0.0.1"
	}

	if nil != net.ParseIP(ip) {
		return ip
	}

	return "127.0.0.1"
}
