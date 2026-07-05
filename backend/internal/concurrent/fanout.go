// Package concurrent は複数 API の並列取得など、パッケージ横断で使う並行処理ヘルパーを提供する。
package concurrent

// Result wraps a fetched value and error for channel-based fan-out patterns.
type Result[T any] struct {
	Data T
	Err  error
}

// FanOut launches a goroutine calling fetch and sends the result to a buffered channel.
func FanOut[T any](fetch func() (T, error)) <-chan Result[T] {
	ch := make(chan Result[T], 1)
	go func() {
		d, e := fetch()
		ch <- Result[T]{d, e}
	}()
	return ch
}
