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
	Repo Repository
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
	started := time.Now()
	r := call.Req(custom)

	if t.Repo != nil {
		// callx reports transport errors as 404 with the error text as body, so keep a bit of the body.
		resp := string(r.Data)
		if len(resp) > 500 {
			resp = resp[:500]
		}
		if err := t.Repo.AddHistory(History{
			Job:        data.Job,
			Project:    data.Project,
			Cron:       data.Cron,
			Method:     custom.Method,
			URL:        custom.URL,
			Status:     r.Code,
			Response:   resp,
			StartedAt:  started,
			DurationMs: time.Since(started).Milliseconds(),
		}); err != nil {
			fmt.Println("[ERROR]", err)
		}
	}
	fmt.Println("[INFO]", time.Now().Format(time.DateTime), custom.Method, custom.URL, r.Code, http.StatusText(r.Code))
}

func NewTask(repo Repository) Task {
	return &task{Repo: repo}
}
