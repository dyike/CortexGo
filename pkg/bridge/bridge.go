package bridge

type NotifyFunc func(topic string, payload string)

var impl NotifyFunc

// SetNotifyImpl 由 main.go 调用，注入 CGO 的实现
func SetNotifyImpl(f NotifyFunc) {
	impl = f
}

// Notify 供 service 层调用，发送事件给 App
func Notify(topic string, payload string) {
	if impl != nil {
		impl(topic, payload)
	}
}
