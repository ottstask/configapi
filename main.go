package main

import (
	_ "github.com/ottstask/configapi/internal/handler"
	"github.com/ottstask/configapi/pkg/meta"

	"github.com/ottstask/configapi/internal/watcher"
	"github.com/ottstask/gofunc"
	"github.com/ottstask/gofunc/pkg/middleware"
)

func main() {
	// mock
	ip := "192.168.2.124"
	watcher.SetValue(meta.IngressKeyPrefix+ip, &meta.IngressConfig{
		HostInfo: map[string]*meta.IngressHostInfo{
			"127.0.0.1:8080": {
				Addr:             "127.0.0.1:8080",
				ConcurrencyLimit: 2,
				QueueSource:      "local://",
			},
		},
	})

	watcher.SetValue(meta.EgressConfigKeyPrefix+ip, &meta.EgressConfig{
		// DomainList:    map[string]bool{"abc": true},
		HostNamespace: map[string]string{"127.0.0.1": "default"},
	})

	watcher.SetValue(meta.DomainConfigKeyPrefix+"abc.default", &meta.DomainConfig{
		Domain: "abc.default",
		IsZero: false,
		DownStreams: map[int]*meta.DownStreamInfo{
			0: {
				Addr:        "127.0.0.1:8080",
				IngressAddr: "127.0.0.1:17000",
			},
		},
	})

	gofunc.Use(middleware.Recover).Use(middleware.Validator)
	gofunc.Serve()
}
