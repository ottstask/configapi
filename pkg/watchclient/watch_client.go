package watchclient

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	json "github.com/goccy/go-json"
	"github.com/ottstask/configapi/internal/meta"
	"github.com/ottstask/configapi/internal/watcher"
	"github.com/r3labs/diff/v3"
)

type clientWatcher struct {
	conn      *websocket.Conn
	value     sync.Map
	initValue sync.Map
	keys      sync.Map
	received  sync.Map

	newKeyCh    chan bool
	url         string
	lastConnect time.Time
}

func NewWatcherClient(url string) (*clientWatcher, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{})
	if err != nil {
		return nil, err
	}
	c := &clientWatcher{conn: conn, url: url,
		value: sync.Map{}, keys: sync.Map{}, received: sync.Map{}, initValue: sync.Map{},
		newKeyCh: make(chan bool, 100)}
	go c.setKeys()
	go c.events()
	return c, nil
}

func (w *clientWatcher) Watch(key string, val meta.Cloneable) error {
	initVal := val.Clone()
	w.keys.Store(key, true)
	w.value.Store(key, initVal)
	w.initValue.Store(key, initVal)
	w.newKeyCh <- true
	return nil
}

func (w *clientWatcher) Load(key string) (any, bool) {
	if _, ok := w.received.Load(key); !ok {
		return nil, false
	}
	return w.value.Load(key)
}

func (w *clientWatcher) Delete(key string) {
	w.value.Delete(key)
	w.keys.Delete(key)
	w.initValue.Delete(key)
	w.received.Delete(key)
	w.newKeyCh <- true
}

func (w *clientWatcher) setKeys() {
	for {
		keys := make([]string, 0)
		<-w.newKeyCh
		for {
			hasNew := false
			select {
			case <-w.newKeyCh:
				hasNew = true
			default:
			}
			if !hasNew {
				break
			}
		}
		w.keys.Range(func(key, value any) bool {
			keys = append(keys, key.(string))
			return true
		})
		key := strings.Join(keys, ",")
		if err := w.conn.WriteMessage(websocket.TextMessage, []byte(key)); err != nil {
			log.Println("set key error", err)
			// TODO: retry/reconnect
		}
	}
}

func (w *clientWatcher) reconnect(err error) {
	for {
		if err != nil {
			log.Println("reconnect for error", err)
		}
		if time.Since(w.lastConnect) < time.Second*5 {
			time.Sleep(time.Second * 5)
		}
		w.lastConnect = time.Now()
		var conn *websocket.Conn
		conn, _, err = websocket.DefaultDialer.Dial(w.url, http.Header{})
		if err != nil {
			continue
		}
		w.conn = conn
		w.newKeyCh <- true
		break
	}
}

func (w *clientWatcher) events() {
	for {
		events := make([]*watcher.Event, 0)
		_, bs, err := w.conn.ReadMessage()
		if err != nil {
			w.reconnect(err)
			continue
		}
		// fmt.Println("recv", string(bs))
		if err := json.Unmarshal(bs, &events); err != nil {
			w.reconnect(err)
			continue
		}
		for _, e := range events {
			if e.Type == watcher.EventTypeAdd || e.Type == watcher.EventTypeUpdate {
				vv, ok := w.initValue.Load(e.Key)
				if !ok {
					log.Println("missing key", e.Key)
					continue
				}
				oldVal := vv.(meta.Cloneable)
				newVal := oldVal.Clone()

				if err := json.Unmarshal(e.JsonValue, newVal); err != nil {
					log.Printf("Unmarshal error %s %v\n", e.Key, err)
					continue
				}
				w.value.Store(e.Key, newVal)
			} else if e.Type == watcher.EventTypePatch {
				vv, ok := w.value.Load(e.Key)
				if !ok {
					continue
				}
				oldVal := vv.(meta.Cloneable)
				newVal := oldVal.Clone()
				diff.Patch(e.Patch, newVal)
				w.value.Store(e.Key, newVal)
			}
			w.received.Store(e.Key, true)
		}
	}
}
