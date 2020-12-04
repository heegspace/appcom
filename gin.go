package appcom

import (
	"fmt"

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
