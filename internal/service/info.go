package service

func GetSystemInfo() any {
	return map[string]any{
		"version": "1.0.0",
		"os":      "android/ios",
	}
}
