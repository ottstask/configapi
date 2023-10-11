package watcher

import (
	"encoding/json"
	"log"
	"reflect"
	"sync"

	"github.com/ottstask/configapi/internal/meta"
	"github.com/r3labs/diff/v3"
)

type EventType string

const EventTypeAdd = "add"
const EventTypeDelete = "delete"
const EventTypeUpdate = "update"
const EventTypePatch = "patch"

var allWatchers = sync.Map{}

type configWatcher struct {
	ch   chan *Event
	keys []string
}

type Event struct {
	Type      EventType      `json:"type"`
	Key       string         `json:"key"`
	JsonValue []byte         `json:"json_value"`
	Patch     diff.Changelog `json:"patch"`
}

var valueMap = sync.Map{}

func GetValue(key string) (any, bool) {
	val, ok := valueMap.Load(key)
	if ok {
		return val, true
	}
	return "", false
}

func SetValue(key string, value meta.Cloneable) {
	val, ok := valueMap.Load(key)
	if ok && reflect.DeepEqual(val, value) {
		return
	}
	newValue := value.Clone()
	valueMap.Store(key, newValue)

	bs, _ := json.Marshal(newValue)

	e := &Event{Key: key, JsonValue: bs, Type: EventTypeAdd}
	if ok {
		e.Type = EventTypePatch
		changelog, err := diff.Diff(val, newValue)
		if err != nil {
			log.Println("cal diff error", err)
			return
		}
		e.Patch = changelog
	}
	emitEvent(e)
}

func DelValue(key string) {
	_, ok := valueMap.Load(key)
	if ok {
		valueMap.Delete(key)
		e := &Event{Type: EventTypeDelete, Key: key}
		emitEvent(e)
	}
}

func emitEvent(e *Event) {
	whs, ok := allWatchers.Load(e.Key)
	if !ok {
		return
	}
	watchers := whs.(*sync.Map)
	watchers.Range(func(key, value interface{}) bool {
		wh := key.(*configWatcher)
		select {
		case wh.ch <- e:
		default:
			log.Println("queue full for watcher", wh.keys)
		}
		return true
	})
}

func NewWatcher() *configWatcher {
	return &configWatcher{ch: make(chan *Event, 1000), keys: []string{}}
}

func (w *configWatcher) SetKeys(keys []string) {
	for _, key := range keys {
		val, ok := allWatchers.Load(key)
		if !ok {
			// TODO: lock to init
			allWatchers.Store(key, &sync.Map{})
			val, _ = allWatchers.Load(key)
		}
		whs := val.(*sync.Map)
		whs.Store(w, true)
	}
	w.keys = keys
}

func (w *configWatcher) Stop() {
	for _, key := range w.keys {
		val, ok := allWatchers.Load(key)
		if !ok {
			continue
		}
		whs := val.(*sync.Map)
		whs.Delete(w)
	}
}

func (w *configWatcher) Events() <-chan *Event {
	return w.ch
}
