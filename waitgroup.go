package waitgroup

import (
	"errors"
	"sync"
)

var (
	ErrInvalidMaxConcurrent = errors.New("maxConcurrent must be greater than 0")
)

type WaitGroup interface {
	Add(delta int)
	Done()
	Wait()
}

type LimitWaitGroup struct {
	wg    sync.WaitGroup
	limit chan struct{}
}

func NewLimitWaitGroup(maxConcurrent int) (WaitGroup, error) {
	if maxConcurrent <= 0 {
		return nil, ErrInvalidMaxConcurrent
	}
	return &LimitWaitGroup{
		limit: make(chan struct{}, maxConcurrent),
	}, nil
}

func (w *LimitWaitGroup) Add(delta int) {
	for i := 0; i < delta; i++ {
		w.limit <- struct{}{}
	}
	w.wg.Add(delta)
}

func (w *LimitWaitGroup) Done() {
	w.wg.Done()
	<-w.limit
}

func (w *LimitWaitGroup) Wait() {
	w.wg.Wait()
}
