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
	urls     map[string]map[string]string
	mux      *sync.RWMutex
	counter  uint64
	filePath string
}

func NewStorage(filePath string) *StorageURL {
	s := &StorageURL{}
	s.mux = &sync.RWMutex{}
	s.urls = make(map[string]map[string]string)
	s.counter = 0
	s.filePath = filePath
	s.restoreFromFile()
	return s
}

func (h *StorageURL) put(user string, key string, value string) {
	_, ok := h.urls[user]
	if !ok {
		h.urls[user] = make(map[string]string)
	}
	h.urls[user][key] = value
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
	User  string `json:"user"`
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
					log.Print("user  = " + event.User)
					log.Print("key   = " + keyStr)
					log.Print("value = " + event.Value)
					h.put(event.User, keyStr, event.Value)
				}
			}
		} else {
			log.Print("can not open file to read: " + err.Error())
		}
	}
}

func (h *StorageURL) PutUserURL(user string, value string) string {
	log.Print("StorageURL.PutURL user=", user)
	key := h.getNewID()

	var event Event
	var data []byte
	var errMarshal error

	if h.filePath != "" {
		event = Event{User: user, Key: key, Value: value}
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
	h.put(user, strKey, value)

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

func (h *StorageURL) GetUserURL(user string, id string) (string, bool) {
	log.Print("StorageURL.GetURL user=", user)
	var val string
	var ok bool
	ok = false

	h.mux.RLock()
	userURLS, isExist := h.urls[user]
	if isExist {
		val, ok = userURLS[id]
	}
	h.mux.RUnlock()
	return val, ok
}

type MyURLS struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *StorageURL) GetUserURLS(user string, url string) ([]byte, bool) {
	log.Print("StorageURL.GetURLS user=", user)
	h.mux.RLock()

	var urls []MyURLS
	var urlsJSON []byte
	retOK := true

	userURLS, ok := h.urls[user]
	if ok {
		for key, element := range userURLS {
			urls = append(urls, MyURLS{ShortURL: url + "/" + key, OriginalURL: element})
		}
	}

	if len(urls) != 0 {
		var err error
		urlsJSON, err = json.Marshal(urls)
		if err != nil {
			log.Print("Marshal all urls fail ", err.Error())
			retOK = false
		}
	}

	h.mux.RUnlock()
	return urlsJSON, retOK
}

func (h *StorageURL) PutURL(value string) string {
	return h.PutUserURL("1", value)
}

func (h *StorageURL) GetURL(id string) (string, bool) {
	return h.GetUserURL("1", id)
}

func (h *StorageURL) GetURLS(url string) ([]byte, bool) {
	return h.GetUserURLS("1", url)
}
