package watchclient

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	json "github.com/goccy/go-json"
	"github.com/kelseyhightower/envconfig"
	"github.com/ottstask/configapi/internal/watcher"
	"github.com/ottstask/configapi/pkg/meta"
	"github.com/r3labs/diff/v3"
)

var watchCfg = &watchConfig{
	Addr: "ws://localhost:8080/api/config-ws",
}

var globalWatcher *clientWatcher

type watchConfig struct {
	Addr string
}

func init() {
	err := envconfig.Process("watch", watchCfg)
	if err != nil {
		log.Fatal(err)
	}
}

type clientWatcher struct {
	conn        *websocket.Conn
	config      sync.Map
	newKeyCh    chan bool
	url         string
	lastConnect time.Time
}

type configValue struct {
	initValue  meta.Cloneable
	value      meta.Cloneable
	received   bool
	callbackCh chan any
}

func InitWatcher() {
	conn, _, err := websocket.DefaultDialer.Dial(watchCfg.Addr, http.Header{})
	if err != nil {
		log.Fatal(err)
	}
	globalWatcher = &clientWatcher{conn: conn, url: watchCfg.Addr, config: sync.Map{}, newKeyCh: make(chan bool, 100)}
	go globalWatcher.setKeys()
	go globalWatcher.recvEvents()
}

func Watch(key string, val meta.Cloneable) <-chan any {
	return globalWatcher.watch(key, val)
}

func Delete(key string) {
	globalWatcher.delete(key)
}

func (w *clientWatcher) watch(key string, val meta.Cloneable) <-chan any {
	cfg := &configValue{
		initValue:  val,
		value:      val.Clone(),
		callbackCh: make(chan any, 100),
	}
	w.config.Store(key, cfg)
	w.newKeyCh <- true
	return cfg.callbackCh
}

func (w *clientWatcher) delete(key string) {
	w.config.Delete(key)
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
		w.config.Range(func(key, value any) bool {
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

func (w *clientWatcher) recvEvents() {
	for {
		events := make([]*watcher.Event, 0)
		_, bs, err := w.conn.ReadMessage()
		if err != nil {
			w.reconnect(err)
			continue
		}
		log.Println("recv event", string(bs))
		if err := json.Unmarshal(bs, &events); err != nil {
			w.reconnect(err)
			continue
		}
		for _, e := range events {
			if e.Type == watcher.EventTypeAdd || e.Type == watcher.EventTypeUpdate {
				vv, ok := w.config.Load(e.Key)
				if !ok {
					log.Println("missing key", e.Key)
					continue
				}
				cfg := vv.(*configValue).Clone()
				newVal := cfg.initValue.Clone()
				if len(e.JsonValue) > 0 {
					if err := json.Unmarshal(e.JsonValue, newVal); err != nil {
						log.Printf("Unmarshal json error %s %v\n", e.Key, err)
						continue
					}
				}
				cfg.value = newVal
				cfg.received = true
				w.newConfig(e.Key, cfg)
			} else if e.Type == watcher.EventTypePatch {
				vv, ok := w.config.Load(e.Key)
				if !ok {
					log.Println("missing key", e.Key)
					continue
				}
				cfg := vv.(*configValue).Clone()
				newVal := cfg.value.Clone()
				diff.Patch(e.Patch, newVal)
				w.newConfig(e.Key, cfg)
			}
		}
	}
}

func (w *clientWatcher) newConfig(key string, cfg *configValue) {
	w.config.Store(key, cfg)
	for {
		hasNew := false
		select {
		case <-cfg.callbackCh:
			hasNew = true
		default:
		}
		if !hasNew {
			break
		}
	}
	cfg.callbackCh <- cfg.value
}

func (c *configValue) Clone() *configValue {
	return &configValue{
		initValue:  c.initValue,
		value:      c.value,
		received:   c.received,
		callbackCh: c.callbackCh,
	}
}
