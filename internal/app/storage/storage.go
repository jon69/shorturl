package storage

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var counter uint64 = 0

func getNewID() string {
	for {
		val := atomic.LoadUint64(&counter)
		if atomic.CompareAndSwapUint64(&counter, val, val+1) {
			str := fmt.Sprint(counter)
			return str
		}
	}
}

type StorageURL struct {
	shorturlMap map[string]string
	mux         *sync.RWMutex
}

func NewStorage() *StorageURL {
	s := &StorageURL{}
	s.mux = &sync.RWMutex{}
	s.shorturlMap = make(map[string]string)
	return s
}

func (h StorageURL) PutURL(value string) string {
	key := getNewID()
	h.mux.Lock()
	defer h.mux.Unlock()
	h.shorturlMap[key] = value
	return key
}

func (h StorageURL) GetURL(id string) (string, bool) {
	h.mux.RLock()
	val, ok := h.shorturlMap[id]
	h.mux.RUnlock()
	return val, ok
}
