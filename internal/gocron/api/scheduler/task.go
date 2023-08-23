package scheduler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prongbang/callx"
	"github.com/prongbang/gocron/internal/pkg/common"
)

type Task interface {
	ApiRequest(data CreateScheduler)
}

type task struct {
}

func (t *task) ApiRequest(data CreateScheduler) {
	header, _ := common.AnyToMap(data.Task.Config.Header)

	c := callx.Config{
		Timeout: 60,
		Interceptor: []callx.Interceptor{
			callx.JSONContentTypeInterceptor(),
		},
	}
	call := callx.New(c)

	custom := callx.Custom{
		URL:    data.Task.Config.URL,
		Method: data.Task.Config.Method,
		Header: header,
		Body:   data.Task.Config.Body,
	}
	r := call.Req(custom)
	fmt.Println("[INFO]", time.Now().Format(time.DateTime), custom.Method, custom.URL, r.Code, http.StatusText(r.Code))
}

func NewTask() Task {
	return &task{}
}
