package watcher

import (
	"encoding/json"
	"log"
	"reflect"
	"sync"

	"github.com/ottstask/configapi/pkg/meta"
	"github.com/r3labs/diff/v3"
)

type EventType string

const EventTypeAdd = "add"
const EventTypeDelete = "delete"
const EventTypeUpdate = "update"
const EventTypePatch = "patch"

var allWatchers = newWatchMap()

type configWatcher struct {
	ch chan *Event
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

	if ok {
		changelog, err := diff.Diff(val, newValue)
		if err != nil {
			log.Println("cal diff error", err)
			return
		}
		e := &Event{Key: key, Type: EventTypePatch, Patch: changelog}
		emitEvent(e)
	} else {
		bs, _ := json.Marshal(newValue)
		e := &Event{Key: key, Type: EventTypePatch, JsonValue: bs}
		emitEvent(e)
	}
}

func DelValue(key string) {
	_, ok := valueMap.Load(key)
	if ok {
		valueMap.Delete(key)
		e := &Event{Type: EventTypeDelete, Key: key}
		emitEvent(e)
	}
}

func NewWatcher() *configWatcher {
	return &configWatcher{ch: make(chan *Event, 1000)}
}

func (w *configWatcher) SetKeys(keys []string) {
	allWatchers.setWatcher(w, keys)
}

func (w *configWatcher) Stop() {
	allWatchers.stopWatcher(w)
}

func (w *configWatcher) Events() <-chan *Event {
	return w.ch
}

func emitEvent(e *Event) {
	whs := allWatchers.getWatcher(e.Key)
	for _, wh := range whs {
		select {
		case wh.ch <- e:
		default:
			log.Println("queue full for watcher")
		}
	}
}
