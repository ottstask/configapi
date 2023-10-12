package watcher

import "sync"

type watchMap struct {
	lock          sync.Mutex
	keyToWatcher  map[string]map[*configWatcher]bool
	watcherToKeys map[*configWatcher][]string
}

func newWatchMap() *watchMap {
	return &watchMap{
		lock:          sync.Mutex{},
		keyToWatcher:  make(map[string]map[*configWatcher]bool),
		watcherToKeys: make(map[*configWatcher][]string),
	}
}

func (w *watchMap) setWatcher(cw *configWatcher, newKeys []string) {
	w.lock.Lock()
	w.watcherToKeys[cw] = newKeys
	for _, k := range newKeys {
		if _, ok := w.keyToWatcher[k]; !ok {
			w.keyToWatcher[k] = make(map[*configWatcher]bool)
		}
		w.keyToWatcher[k][cw] = true
	}
	w.lock.Unlock()
}

func (w *watchMap) getWatcher(key string) []*configWatcher {
	w.lock.Lock()
	if val, ok := w.keyToWatcher[key]; ok {
		ret := make([]*configWatcher, len(val))
		i := 0
		for k := range val {
			ret[i] = k
			i++
		}
		w.lock.Unlock()
		return ret
	}
	w.lock.Unlock()
	return nil
}

func (w *watchMap) stopWatcher(cw *configWatcher) {
	w.lock.Lock()
	if keys, ok := w.watcherToKeys[cw]; ok {
		for _, k := range keys {
			if val, ok := w.keyToWatcher[k]; ok {
				delete(val, cw)
			}
			if len(w.keyToWatcher[k]) == 0 {
				delete(w.keyToWatcher, k)
			}
		}
		delete(w.watcherToKeys, cw)
	}
	w.lock.Unlock()
}
