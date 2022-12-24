package storage

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

type StorageURL struct {
	shorturlMap map[string]string
	mux         *sync.RWMutex
	counter     uint64
}

func NewStorage() *StorageURL {
	s := &StorageURL{}
	s.mux = &sync.RWMutex{}
	s.shorturlMap = make(map[string]string)
	s.counter = 0
	return s
}

func (h *StorageURL) getNewID() string {
	log.Print(h.counter)
	for {
		val := atomic.LoadUint64(&h.counter)
		if atomic.CompareAndSwapUint64(&h.counter, val, val+1) {
			str := fmt.Sprint(h.counter)
			return str
		}
	}
}

func (h *StorageURL) PutURL(value string) string {
	key := h.getNewID()
	h.mux.Lock()
	defer h.mux.Unlock()
	h.shorturlMap[key] = value
	return key
}

func (h *StorageURL) GetURL(id string) (string, bool) {
	h.mux.RLock()
	val, ok := h.shorturlMap[id]
	h.mux.RUnlock()
	return val, ok
}
