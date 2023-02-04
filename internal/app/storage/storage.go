package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

type StorageURL struct {
	shorturlMap map[string]string
	mux         *sync.RWMutex
	counter     uint64
	filePath    string
}

func NewStorage(filePath string) *StorageURL {
	s := &StorageURL{}
	s.mux = &sync.RWMutex{}
	s.shorturlMap = make(map[string]string)
	s.counter = 0
	s.filePath = filePath
	s.restoreFromFile()
	return s
}

func (h *StorageURL) getNewID() uint64 {
	log.Print(h.counter)
	for {
		val := atomic.LoadUint64(&h.counter)
		if atomic.CompareAndSwapUint64(&h.counter, val, val+1) {
			return h.counter
		}
	}
}

type Event struct {
	Key   uint64 `json:"key"`
	Value string `json:"value"`
}

func max(value1 uint64, value2 uint64) uint64 {
	if value1 > value2 {
		return value1
	}
	return value2
}

func (h *StorageURL) restoreFromFile() {
	if h.filePath != "" {
		log.Print("opening file...")
		file, err := os.OpenFile(h.filePath, os.O_RDONLY|os.O_CREATE, 0777)
		if err == nil {
			defer file.Close()
			log.Print("readin from file...")
			reader := bufio.NewReader(file)
			for data, err := reader.ReadBytes('\n'); err == nil; data, err = reader.ReadBytes('\n') {
				event := Event{}
				err = json.Unmarshal(data, &event)
				if err == nil {
					h.counter = max(h.counter, event.Key)
					keyStr := fmt.Sprint(event.Key)
					log.Print("key = " + keyStr)
					log.Print("value = " + event.Value)
					h.shorturlMap[keyStr] = event.Value
				}
			}
		} else {
			log.Print("can not open file to read: " + err.Error())
		}
	}
}

func (h *StorageURL) PutURL(value string) string {
	log.Print("StorageURL.PutURL")
	key := h.getNewID()

	var event Event
	var data []byte
	var errMarshal error

	if h.filePath != "" {
		event = Event{Key: key, Value: value}
		data, errMarshal = json.Marshal(&event)
		if errMarshal != nil {
			log.Print("can not json.marshal")
		} else {
			data = append(data, '\n')
		}
	}

	h.mux.Lock()
	defer h.mux.Unlock()

	strKey := fmt.Sprint(key)
	h.shorturlMap[strKey] = value

	if h.filePath != "" && errMarshal == nil {
		log.Print("opening file...")
		file, err := os.OpenFile(h.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
		if err == nil {
			defer file.Close()
			log.Print("writing to file...")
			log.Print(data)
			_, err = file.Write(data)
			if err != nil {
				log.Print("can not write to file")
			}
		} else {
			log.Print("can not open file to write: " + err.Error())
		}
	}
	return strKey
}

func (h *StorageURL) GetURL(id string) (string, bool) {
	log.Print("StorageURL.GetURL")
	h.mux.RLock()
	val, ok := h.shorturlMap[id]
	h.mux.RUnlock()
	return val, ok
}

type MyURLS struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *StorageURL) GetURLS() ([]byte, bool) {
	log.Print("StorageURL.GetURLS")
	h.mux.RLock()

	var urls []MyURLS
	for key, element := range h.shorturlMap {
		urls = append(urls, MyURLS{ShortURL: key, OriginalURL: element})
	}

	if len(urls) == 0 {
		return []byte{}, true
	}

	urlsJSON, err := json.Marshal(urls)
	if err != nil {
		log.Print("Marshal all urls fail ", err.Error())
		return []byte{}, false
	}

	h.mux.RUnlock()
	return urlsJSON, true
}
