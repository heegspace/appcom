package nodecom

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

var ERR_CODE map[string]interface{}

// 加载响应码和响应信息文件
// @param file 配置文件
func LoadCodeFromFile(file string) {
	if nil != ERR_CODE {
		return
	}

	f, err := os.Open(file)
	if err != nil {
		fmt.Println(err)

		return
	}
	defer f.Close()

	data, _ := ioutil.ReadAll(f)

	err = json.Unmarshal([]byte(data), &ERR_CODE)
	if nil != err {
		fmt.Println(err)
		return
	}

	return
}

// 获取响应的状态码
// @param enum 	枚举参数
// @return float64
func ResponseCode(enum string) float64 {
	if _, ok := ERR_CODE[enum]; !ok {
		return -1
	}

	cstr := ERR_CODE[enum].(map[string]interface{})

	if _, ok := cstr["code"]; !ok {
		return -1
	}

	return cstr["code"].(float64)
}

// 获取响应中对应状态码的信息
// @param enum 	枚举参数
// @return string
func ResponseMsg(enum string) string {
	if _, ok := ERR_CODE[enum]; !ok {
		return "Didn't error str!"
	}

	cstr := ERR_CODE[enum].(map[string]interface{})

	if _, ok := cstr["str"]; !ok {
		return "Didn't error str!"
	}

	return cstr["str"].(string)
}
