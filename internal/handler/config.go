package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ottstask/configapi/internal/watcher"
	"github.com/ottstask/gofunc"
	"github.com/ottstask/gofunc/pkg/websocket"

	json "github.com/goccy/go-json"
)

var jsonEncoder = json.Marshal

func init() {
	gofunc.Handle(&configHandler{})
}

type GetConfigRequest struct {
	Key string `schema:"key" validate:"required"`
}

type GetConfigResponse struct {
	Values map[string][]byte
}

type WatchConfigRequest struct {
	Key string `schema:"key" validate:"required"`
}

type configHandler struct{}

func (h *configHandler) Get(ctx context.Context, req *GetConfigRequest, rsp *GetConfigResponse) error {
	keys := strings.Split(req.Key, ",")
	for _, key := range keys {
		val, _ := watcher.GetValue(key)
		var value []byte
		if val != nil {
			value, _ = jsonEncoder(val)
		}
		rsp.Values[key] = value
	}
	return nil
}

func (s *configHandler) Stream(ctx context.Context, req websocket.RecvStream, rsp websocket.SendStream) error {
	watch := watcher.NewWatcher()
	defer watch.Stop()

	// recv init keys
	var keys []string
	var recvFunc = func() error {
		bs, err := req.Recv()
		if err != nil {
			return fmt.Errorf("recv key error: %v", err)
		}
		keys = strings.Split(string(bs), ",")
		watch.SetKeys(keys)
		return nil
	}
	if err := recvFunc(); err != nil {
		return err
	}

	// init response value
	initValues := make([]*watcher.Event, 0)
	for _, key := range keys {
		val, _ := watcher.GetValue(key)
		bs, _ := jsonEncoder(val)
		initValues = append(initValues, &watcher.Event{JsonValue: bs, Key: key, Type: watcher.EventTypeAdd})
	}
	bs, _ := jsonEncoder(initValues)
	if err := rsp.Send(bs); err != nil {
		return fmt.Errorf("send error: %v", err)
	}

	// check watch key update
	errChan := make(chan error, 2)
	go func() {
		for {
			err := recvFunc()
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	// send key change
	go func() {
		for e := range watch.Events() {
			events := make([]*watcher.Event, 0)
			events = append(events, e)
			for {
				hasEvent := false
				select {
				case e := <-watch.Events():
					hasEvent = true
					events = append(events, e)
				default:
				}
				if !hasEvent {
					break
				}
			}
			bs, _ := jsonEncoder(&events)
			if err := rsp.Send(bs); err != nil {
				errChan <- err
				return
			}

		}
	}()

	err := <-errChan
	return err
}
