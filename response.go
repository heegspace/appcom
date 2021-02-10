package appcom

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    data,
	})
}

func HandleErr(c *gin.Context, code float64, msg string, err error) {
	c.JSON(http.StatusOK, gin.H{
		"code":    code,
		"message": msg,
		"error":   err,
	})
}

// 写入加密数据
//
// @param c
// @param crypt 	加密的数据
// @param style 	加密方式[public/private]
// 		public 		公共加解密
// 		private		私有加解密
// @param key 		加密的key
//
func HandleEnc(c *gin.Context, crypt string, style string, key string) {
	// X-Crypt-Style
	c.Writer.Header().Set("SE8DF5B93A6EFCEC229845238CB3F6412", style)
	// X-Crypt-Key
	c.Writer.Header().Set("K466B0BD10CD3C6CB55D541F3D4585CA1", key)

	c.String(http.StatusOK, crypt)

	return
}

// 自定义头部和状态码响应数据
//
// @param c
// @param headers 	响应头
// @param crypt 	加密的数据
// @param code 		响应状态码
//
func HandleCustomEnc(c *gin.Context, headers map[string]string, crypt string, code int) {
	for k, v := range headers {
		c.Writer.Header().Set(k, v)
	}

	c.String(code, crypt)
	return
}
