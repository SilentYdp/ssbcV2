package network

import "net/http"

const BadRequest = "Request Param Invalid"

//解析出Get请求中的指定参数
func ParseGetParam(key string, request *http.Request) string {
	params := request.URL.Query()
	v := params.Get(key)
	return v
}

//返回请求参数错误
func BadRequestResponse(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusBadRequest)
	_, _ = writer.Write([]byte(BadRequest))
}

//解析出post请求的body
func ParsePostBody(writer http.ResponseWriter, request *http.Request) {

}
