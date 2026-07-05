package concurrent

import (
	"errors"
	"testing"
)

func TestFanOut_Success(t *testing.T) {
	ch := FanOut(func() (int, error) { return 42, nil })
	r := <-ch
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.Data != 42 {
		t.Errorf("Data = %d, want 42", r.Data)
	}
}

func TestFanOut_Error(t *testing.T) {
	wantErr := errors.New("fetch failed")
	ch := FanOut(func() ([]string, error) { return nil, wantErr })
	r := <-ch
	if !errors.Is(r.Err, wantErr) {
		t.Errorf("Err = %v, want %v", r.Err, wantErr)
	}
	if r.Data != nil {
		t.Errorf("Data = %v, want nil", r.Data)
	}
}

// FanOut はバッファ付きチャネルを返すため、受信されなくても goroutine がリークしない
func TestFanOut_NonBlocking(t *testing.T) {
	done := make(chan struct{})
	FanOut(func() (int, error) {
		defer close(done)
		return 1, nil
	})
	<-done // 受信者不在でも fetch goroutine が完了できる
}
