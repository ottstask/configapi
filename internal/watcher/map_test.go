package watcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatchMap(t *testing.T) {
	m := newWatchMap()
	cw := &configWatcher{}

	m.setWatcher(cw, []string{"k1"})
	ws := m.getWatcher("k1")
	assert.Equal(t, len(ws), 1)
	assert.Equal(t, ws[0], cw)

	m.setWatcher(cw, []string{"k1", "k2"})
	ws = m.getWatcher("k2")
	assert.Equal(t, len(ws), 1)
	assert.Equal(t, ws[0], cw)

	cw2 := &configWatcher{}

	m.setWatcher(cw2, []string{"k1", "k2"})
	ws = m.getWatcher("k2")
	assert.Equal(t, len(ws), 2)

	m.stopWatcher(cw2)
	ws = m.getWatcher("k2")
	assert.Equal(t, len(ws), 1)
	assert.Equal(t, ws[0], cw)

	m.stopWatcher(cw)
	ws = m.getWatcher("k2")
	assert.Equal(t, len(ws), 0)
}
