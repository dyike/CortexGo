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
	"unsafe"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/pkg/bridge"
)

var globalCallback C.EventCallback

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
func InitSDK(workDir *C.char, configJson *C.char) *C.char {
	dir := C.GoString(workDir)
	cfg := C.GoString(configJson)

	err := config.Initialize(dir, cfg)
	if err != nil {
		return C.CString("Error: " + err.Error())
	}
	return C.CString("Success")
}

//export RegisterCallback
func RegisterCallback(cb C.EventCallback) {
	globalCallback = cb
}

//export UpdateConfig
func UpdateConfig(jsonStr *C.char) *C.char {
	newCfg := C.GoString(jsonStr)
	config.Update(newCfg)
	return C.CString("Success")
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
