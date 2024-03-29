package appcom

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"net/http"
)

// gin中设置跨域的中间件
func Corss() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			fmt.Println("origin-->", origin)
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Origin", origin) // 这是允许访问所有域
			// c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE, HEAD") //服务器支持的所有跨域请求的方法,为了避免浏览次请求的多次'预检'请求
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma,Tag-Unid,My-Id,B-Idden,Extra-Fname")
			c.Header("Access-Control-Expose-Headers", "SE8DF5B93A6EFCEC229845238CB3F6412,K466B0BD10CD3C6CB55D541F3D4585CA1,Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar,Extra-Fname") // 跨域关键设置 让浏览器可以解析
			c.Header("Access-Control-Max-Age", "172800")                                                                                                                                                                                                                                           // 缓存请求信息 单位为秒
			c.Header("Access-Control-Allow-Credentials", "true")                                                                                                                                                                                                                                   //  跨域请求是否需要带cookie信息 默认设置为true
		}

		if method == "OPTIONS" {
			c.String(http.StatusOK, "Options Request!")
		}

		c.Next()
	}
}
