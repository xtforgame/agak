package requestsender

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

func DefaultResponseValidator(result *RequestResult) error {
	return nil
	// if result.Response.StatusCode < 200 || result.Response.StatusCode >= 400 {
	// 	return errors.New("status code : " + fmt.Sprintf("%d", result.Response.StatusCode))
	// }
	// return nil
}

func BasicResponseValidator(result *RequestResult) error {
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 400 {
		return errors.New("status code : " + fmt.Sprintf("%d", result.Response.StatusCode))
	}
	return nil
}

type RequestSenderClient struct {
	proxyUrl string
	proxy    *url.URL
	tr       *http.Transport
	client   *http.Client
}

func NewRequestSenderClient(proxyUrl string) *RequestSenderClient {
	// proxyUrl := "http://115.215.71.12:808"
	var proxy *url.URL
	var httpProxyURL func(*http.Request) (*url.URL, error)
	if proxyUrl != "" {
		proxy, _ := url.Parse(proxyUrl)
		httpProxyURL = http.ProxyURL(proxy)
	}
	tr := &http.Transport{
		Proxy:           httpProxyURL,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
	}
	return &RequestSenderClient{
		proxyUrl: proxyUrl,
		proxy:    proxy,
		tr:       tr,
		client:   client,
	}
}

type RequestConfig struct {
	Method    string
	Url       string
	Header    map[string]string
	Body      []byte
	Form      url.Values
	Validator ResponseValidator
}

func (client *RequestSenderClient) SendRequest(ctx context.Context, config *RequestConfig) (*RequestResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	method := config.Method
	if method == "" {
		method = "GET"
	}

	validator := config.Validator
	if validator == nil {
		validator = DefaultResponseValidator
	}

	var body io.Reader
	if config.Body != nil {
		body = bytes.NewBuffer(config.Body)
	} else if config.Form != nil {
		body = strings.NewReader(config.Form.Encode())
	}
	request, _ := http.NewRequest(method, config.Url, body)
	if config.Header != nil {
		for k, v := range config.Header {
			request.Header.Set(k, v)
		}
	}
	var result *RequestResult
	var mu sync.Mutex

	request.WithContext(ctx)
	f := func(resp *http.Response, err error) error {
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil || body == nil {
			body = []byte{}
		}
		r := &RequestResult{
			Response: resp,
			Header:   resp.Header,
			Body:     body,
		}
		err = validator(r)
		if err == nil {
			mu.Lock()
			if result == nil {
				result = r
			}
			mu.Unlock()
			return nil
		}
		return err
	}
	c := make(chan error, 1)
	go func() {
		c <- f(client.client.Do(request))
	}()

	var cancelErr error
	for {
		select {
		case <-ctx.Done():
			// fmt.Println("XCancelling request")
			cancelErr = errors.New("request canceled")
			client.tr.CancelRequest(request)
		case err := <-c:
			if err != nil {
				// fmt.Println("Error", err)
			}
			if cancelErr != nil {
				return result, cancelErr
			}
			return result, err
		}
	}
}

type TaskRunnerOptions struct {
	Owner               *RequestSender
	RequestInterval     time.Duration
	WaitForPreviousTask bool

	ShouldRun    func(*RequestTask, int) bool
	GetDelayTime func(*RequestTask, int) time.Duration
	GetCache     func(ctx context.Context, task *RequestTask, options TaskRunnerOptions) *TaskResult

	HandleTaskResult func(*RequestTask, int, *TaskResult, chan *TaskResult, chan bool)

	Retry      int
	RetryDelay time.Duration
}

type RequestTask struct {
	TaskRunnerOptions *TaskRunnerOptions

	Index        int
	Name         string
	ResourceName string

	RequestConfig *RequestConfig

	UseProxy   bool
	Threads    int
	ExpiryTime time.Duration

	Transform             func(*RequestTask, int, *TaskResult)
	TransformFromResponse func(*RequestTask, int, *TaskResult)
	TransformFromCache    func(*RequestTask, int, *TaskResult)
	HandleTaskResult      func(*RequestTask, int, *TaskResult, chan *TaskResult, chan bool)

	CustomData map[string]interface{}
}

type TaskResult struct {
	Task          *RequestTask
	RequestResult *RequestResult
	Errors        []error
	IsFromCache   bool

	CustomData map[string]interface{}
}

type RequestSender struct {
	client       *RequestSenderClient
	proxyClients []*RequestSenderClient
}

func NewRequestSender(proxyUrls []string) *RequestSender {
	reqSender := &RequestSender{
		client:       NewRequestSenderClient(""),
		proxyClients: make([]*RequestSenderClient, len(proxyUrls)),
	}
	for i, proxyUrl := range proxyUrls {
		reqSender.proxyClients[i] = NewRequestSenderClient(proxyUrl)
	}
	return reqSender
}

func (reqSender *RequestSender) SendRequest(ctx context.Context, config *RequestConfig) (*RequestResult, error) {
	return reqSender.client.SendRequest(ctx, config)
}

func (reqSender *RequestSender) SendRequestByMultipleProxies(
	ctx context.Context,
	config *RequestConfig,
	threads int,
	expiryTime time.Duration,
) (*RequestResult, []error) {
	if ctx == nil {
		ctx = context.Background()
	}
	from := 0
	to := threads
	max := len(reqSender.proxyClients)
	if threads < 0 {
		from = max + threads
		to = max
		if from < 0 {
			from = 0
		}
		threads = -threads
	}
	if to > max || to == 0 {
		to = max
	}
	threads = to - from
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var result *RequestResult
	resultErr := make([]error, threads+1)
	errChan := make(chan error, threads)
	errCounter := 0
	var wg sync.WaitGroup
	wg.Add(threads + 1)
	for index := from; index < to; index++ {
		f := from
		i := index
		go func() {
			defer wg.Done()
			r, err := reqSender.proxyClients[i].SendRequest(ctx, config)
			if err != nil {
				mu.Lock()
				resultErr[i-f] = err
				errChan <- err
				mu.Unlock()
			} else {
				mu.Lock()
				if result == nil {
					result = r
				}
				mu.Unlock()
				cancel()
			}
		}()
	}
	if expiryTime == 0 {
		expiryTime = 36000 * time.Second
	}
	go func() {
		sleep := make(chan struct{}, 1)
		go func() {
			time.Sleep(expiryTime)
			sleep <- struct{}{}
		}()
		defer wg.Done()
		select {
		case <-errChan:
			mu.Lock()
			errCounter++
			if errCounter >= threads {
				mu.Unlock()
				return
			}
			mu.Unlock()
		case <-sleep:
			mu.Lock()
			if result == nil {
				resultErr[threads] = errors.New("timeout")
			}
			mu.Unlock()
			cancel()
		case <-ctx.Done():
			return
		}
	}()
	start := time.Now()
	wg.Wait()
	elapsed := time.Since(start)
	printElapsedTime(elapsed)
	return result, resultErr
}

func printElapsedTime(elapsed time.Duration) {
	// fmt.Println("elapsed :", elapsed)
}

func (reqSender *RequestSender) RunTask(
	ctx context.Context,
	task *RequestTask,
	options TaskRunnerOptions,
) *TaskResult {
	result := &TaskResult{}
	if task.RequestConfig.Url == "" {
		result.Errors = []error{errors.New("url is empty(task: " + task.Name + ")")}
	}

	if task.UseProxy {
		result.RequestResult, result.Errors = reqSender.SendRequestByMultipleProxies(
			ctx,
			task.RequestConfig,
			task.Threads,
			task.ExpiryTime,
		)
	} else {
		result.Errors = []error{nil}
		result.RequestResult, result.Errors[0] = reqSender.SendRequest(
			ctx,
			task.RequestConfig,
		)
	}
	return result
}

type TaskResultsWatcher struct {
	TaskResultChan chan *TaskResult
	CancelChan     chan bool
	FinishChan     chan bool
}

func (watcher *TaskResultsWatcher) Watch(
	handle func(*TaskResult),
) {
	done := false
	for !done {
		select {
		case r := <-watcher.TaskResultChan:
			{
				handle(r)
				break
			}
		case <-watcher.FinishChan:
			{
				done = true
				break
			}
		}
	}
}

func (reqSender *RequestSender) RunTasks(
	tasks []*RequestTask,
	options TaskRunnerOptions,
) *TaskResultsWatcher {
	taskResultChan := make(chan *TaskResult, 0)
	cancelChan := make(chan bool, 0)
	finishChan := make(chan bool, 0)
	stopped := false
	go func() {
		<-cancelChan
		stopped = true
	}()

	options.Owner = reqSender
	shouldRun := options.ShouldRun
	if shouldRun == nil {
		shouldRun = func(*RequestTask, int) bool { return true }
	}

	getDelayTime := options.GetDelayTime
	if getDelayTime == nil {
		getDelayTime = func(task *RequestTask, taskIndex int) time.Duration { return task.TaskRunnerOptions.RequestInterval }
	}

	getCache := options.GetCache
	if getCache == nil {
		getCache = func(ctx context.Context, task *RequestTask, options TaskRunnerOptions) *TaskResult { return nil }
	}

	handleTaskResult := options.HandleTaskResult
	if handleTaskResult == nil {
		handleTaskResult = func(task *RequestTask, taskIndex int, taskResult *TaskResult, taskResultChan chan *TaskResult, cancelChan chan bool) {
			taskResultChan <- taskResult
		}
	}

	finish := func() {
		close(cancelChan)
		close(finishChan)
		stopped = true
	}

	go func() {
		var wg sync.WaitGroup
		for i, t := range tasks {
			t.Index = i
			t.TaskRunnerOptions = &options
			if stopped == true || !shouldRun(t, i) {
				break
			}
			run := func() {
				task := t
				defer wg.Done()
				result := getCache(nil, task, options)
				if result == nil {
					result = reqSender.RunTask(nil, task, options)
					if task.TransformFromResponse != nil {
						task.TransformFromResponse(task, i, result)
					}
					result.IsFromCache = false
				} else {
					if task.TransformFromCache != nil {
						task.TransformFromCache(task, i, result)
					}
					result.IsFromCache = true
				}
				if task.Transform != nil {
					task.Transform(task, i, result)
				}
				result.Task = task
				if task.HandleTaskResult != nil {
					task.HandleTaskResult(task, i, result, taskResultChan, cancelChan)
				} else {
					handleTaskResult(task, i, result, taskResultChan, cancelChan)
				}
			}
			if options.WaitForPreviousTask {
				wg.Add(1)
				run()
			} else {
				wg.Add(1)
				go run()
			}
			if i == len(tasks)-1 {
				break
			}
			time.Sleep(getDelayTime(t, i))
		}
		wg.Wait()
		finish()
	}()
	return &TaskResultsWatcher{
		TaskResultChan: taskResultChan,
		CancelChan:     cancelChan,
		FinishChan:     finishChan,
	}
}
