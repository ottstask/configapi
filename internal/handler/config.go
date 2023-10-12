package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ottstask/configapi/internal/watcher"
	"github.com/ottstask/configapi/pkg/meta"
	"github.com/ottstask/gofunc"
	"github.com/ottstask/gofunc/pkg/ecode"
	"github.com/ottstask/gofunc/pkg/websocket"

	json "github.com/goccy/go-json"
)

var jsonEncoder = json.Marshal

func init() {
	gofunc.Handle(&configHandler{})
}

type GetConfigRequest struct {
	Domain string `schema:"domain" validate:"required"`
}

type WatchConfigRequest struct {
	Key string `schema:"key" validate:"required"`
}

type configHandler struct{}

func (h *configHandler) GetDomain(ctx context.Context, req *GetConfigRequest, rsp *meta.DomainConfig) error {
	key := meta.DomainConfigKeyPrefix + req.Domain
	val, ok := watcher.GetValue(key)
	if !ok {
		return ecode.Errorf(404, "domain config for %s not found", req.Domain)
	}
	*rsp = *(val.(*meta.DomainConfig))
	return nil
}

func (s *configHandler) Stream(ctx context.Context, req websocket.RecvStream, rsp websocket.SendStream) error {
	watch := watcher.NewWatcher()
	defer watch.Stop()

	// check watch key update
	errChan := make(chan error, 2)
	keysChan := make(chan []string, 100)
	go func() {
		// currKeys := make(map[string]bool)
		for {
			bs, err := req.Recv()
			if err != nil {
				errChan <- fmt.Errorf("recv key error: %v", err)
				return
			}
			keys := strings.Split(string(bs), ",")
			watch.SetKeys(keys)
			keysChan <- keys
		}
	}()

	// send key change
	go func() {
		currKeys := make(map[string]bool)
		for {
			events := make([]*watcher.Event, 0)
			select {
			case e := <-watch.Events():
				events = append(events, e)
			case keys := <-keysChan:
				// Add init event
				newKeys := make(map[string]bool)
				for _, k := range keys {
					if !currKeys[k] {
						bs := []byte("")
						val, ok := watcher.GetValue(k)
						if ok {
							bs, _ = jsonEncoder(val)
						}
						events = append(events, &watcher.Event{JsonValue: bs, Key: k, Type: watcher.EventTypeAdd})
					}
					newKeys[k] = true
				}
				currKeys = newKeys
			}

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
