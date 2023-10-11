package main

import (
	"time"

	_ "github.com/ottstask/configapi/internal/handler"
	"github.com/ottstask/configapi/internal/meta"

	"github.com/ottstask/configapi/internal/watcher"
	"github.com/ottstask/gofunc"
	"github.com/ottstask/gofunc/pkg/middleware"
)

func main() {
	val := &meta.MetaObject{
		Services: map[string]*meta.ServiceInfo{
			"abc":  {Name: "def"},
			"abcd": {Name: "deff"},
		},
	}
	watcher.SetValue("meta_object", val)
	go func() {
		time.Sleep(time.Second * 5)
		val.Services["abc"].Name = "kkk"
		watcher.SetValue("meta_object", val)
	}()

	gofunc.Use(middleware.Recover).Use(middleware.Validator)
	gofunc.Serve()
}
