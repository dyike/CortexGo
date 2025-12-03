package main

import (
	"encoding/json"

	"github.com/dyike/CortexGo/internal/service"
)

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

func Dispatch(method string, paramsJson string) string {
	var result any
	var err error

	switch method {
	case "auth.login":
		result, err = service.Login(paramsJson)
	case "system.info":
		result = service.GetSystemInfo()
	default:
		return jsonResp(404, "Method not found", nil)
	}
	if err != nil {
		return jsonResp(500, err.Error(), nil)
	}
	return jsonResp(200, "Ok", result)
}

func jsonResp(code int, msg string, data any) string {
	resp := Response{Code: code, Msg: msg, Data: data}
	b, _ := json.Marshal(resp)
	return string(b)
}
