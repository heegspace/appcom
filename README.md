# appcom
app服务公用部分

# 接口
## NeedLogin
```
/**
* 权限中间件，主要是确认是否登陆成功
*
* @param callback 	主要是在每个服务中将用户ID传递出去
* @param timeout 	token的过期时间
*/ 
func NeedLogin(callback func(uid string) bool,timeout int64) gin.HandlerFunc{}
```

## 编码Token
```
/**
* 编码token
* 
* @param src 	产生token的用户信息
* @param key 	产生token的秘钥
*
* @return token 产生以后的token信息
* @return err 	产生token是否出错
*/
func EnCookie(src TokenInfo, key string) (token string,err error)
```

## 解码Token
```
/**
* 解码token
*
* @param  src 	token信息
* @param  key 	解码token的秘钥
*
* @return token  解码后的token用户信息
* @return err    解码状态
*/
func DeCookie(src string, key string) (token TokenInfo,err error) 
```

## 装载错误码文件
```
/**
* 装载错误码文件
*
* @param file 	错误码文件路径
*/
func LoadCodeFromFile(file string) 
```

## 获取错误码
通过编码从错误码配置文件中获取错误码
```
func ResponseCode(enum string) float64
```

## 获取错误信息
通过编码从错误码配置文件中获取错误信息
```
func ResponseMsg(enum string) string
```

## 请求的响应函数
### 成功
```
/**
* @param data 	响应的数据
*/
func HandleOK(c *gin.Context, data interface{})

响应格式是：
	{
		"code": 200,
		"message": "success",
		"data": data
	}
```
### 失败
```
/**
*
* @param code 	响应的状态码
* @param msg 	响应的错误信息
*/
func HandleErr(c *gin.Context, code float64, msg string, err error)

响应格式是：
	{
		"code": code,
		"message", msg,
		"error": err,
	}
```
