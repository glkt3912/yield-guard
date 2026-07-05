package concurrent

import "fmt"

// SafeCall は fn を実行し、panic を error に変換して返す。
// errgroup 配下の goroutine で panic すると Wait がプロセス全体に再送出するため、
// タイル単位の部分失敗として扱いたい呼び出しはこれで包む。
func SafeCall(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}
