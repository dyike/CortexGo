package main

/*
#include <stdlib.h>

// 定义回调函数的函数指针类型
// topic: event
// payload: JSON数据
typedef void (*EventCallback)(char* topic, char* payload);

// 声明一个帮助函数来调用回调（Go 不能直接调用 C 函数指针，需通过 C 桥接）
static void invokeCallback(EventCallback cb, char* topic, char* payload) {
    if (cb) {
        cb(topic, payload);
    }
}
*/
import "C"
import (
	"encoding/json"
	"unsafe"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/pkg/app"
	"github.com/dyike/CortexGo/pkg/bridge"
)

var globalCallback C.EventCallback
var appRuntime *app.Runtime

func init() {
	// 设置Go内部发送事件的实现
	bridge.SetNotifyImpl(func(topic, payload string) {
		if globalCallback == nil {
			return
		}
		cTopic := C.CString(topic)
		cPayload := C.CString(payload)
		// 必须释放 Go 创建的 C 字符串
		defer C.free(unsafe.Pointer(cTopic))
		defer C.free(unsafe.Pointer(cPayload))

		C.invokeCallback(globalCallback, cTopic, cPayload)
	})
}

//export InitSDK
func InitSDK(configPath *C.char) *C.char {
	path := C.GoString(configPath)

	if err := config.Initialize(path); err != nil {
		return C.CString("Error: " + err.Error())
	}

	if appRuntime != nil {
		appRuntime.Close()
	}

	rt, err := app.NewRuntime(config.DefaultManager(), app.WithNotifier(bridge.Notify))
	if err != nil {
		return C.CString("Error: " + err.Error())
	}
	appRuntime = rt
	return C.CString("Success")
}

//export RegisterCallback
func RegisterCallback(cb C.EventCallback) {
	globalCallback = cb
}

//export UpdateConfig
func UpdateConfig(jsonStr *C.char) *C.char {
	newCfg := C.GoString(jsonStr)
	if err := config.Update(newCfg); err != nil {
		return C.CString("Error: " + err.Error())
	}
	return C.CString("Success")
}

//export GetConfig
func GetConfig() *C.char {
	cfg := config.Get()
	b, _ := json.Marshal(cfg)
	return C.CString(string(b))
}

//export Call
func Call(method *C.char, params *C.char) *C.char {
	m := C.GoString(method)
	p := C.GoString(params)
	resp := Dispatch(m, p)
	return C.CString(resp)
}

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

func main() {}
