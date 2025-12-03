package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dyike/CortexGo/pkg/bridge"
)

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(jsonStr string) (any, error) {
	var p LoginParams
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return nil, fmt.Errorf("invalid params")
	}

	// 模拟耗时操作
	// 实际项目中，这里可能会调用 HTTP 请求
	time.Sleep(500 * time.Millisecond)

	if p.Username == "admin" {
		// 模拟：登录成功后，主动给 App 发送一个事件（例如 Socket 连接成功）
		go func() {
			time.Sleep(1 * time.Second)
			bridge.Notify("socket.status", `{"connected": true}`)
		}()

		return map[string]string{
			"token": "token_123456",
			"uid":   "999",
		}, nil
	}

	return nil, fmt.Errorf("wrong password")
}

func GetSystemInfo() any {
	return map[string]any{
		"version": "1.0.0",
		"os":      "android/ios",
	}
}
