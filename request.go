package appcom

import (
	"io/ioutil"
	"mime/multipart"

	"github.com/gin-gonic/gin"
)

type RequestPage struct {
	Page int `form:"page"`
	Size int `form:"size"`
}

// 从请求中获取page和size，主要是获取请求区间
func PageSize(c *gin.Context) RequestPage {
	var param RequestPage

	err := c.ShouldBindQuery(&param)
	if err != nil {
		param.Page = 0
		param.Size = 20

		return param
	}

	if 0 >= param.Page {
		param.Page = 0
	}

	if 0 >= param.Size {
		param.Size = 20
	}

	return param
}

// 读取http中POST的form中文件的数据
//
// @param file 	对就对象
//
func ReadFormFileData(file *multipart.FileHeader) (data []byte, err error) {
	refile, err := file.Open()
	if nil != err {
		return
	}
	defer refile.Close()

	data, err = ioutil.ReadAll(refile)
	if nil != err {
		return
	}

	return
}
