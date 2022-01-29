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

	ShouldRun         func(*RequestTask, int) bool
	GetDelayTime      func(*RequestTask, int, *TaskRunnerOptions, *TaskResult) time.Duration
	TryEarlyDone      func(*RequestTask, int, *TaskRunnerOptions, *TaskResult) bool
	GetCacheOrVirtual func(ctx context.Context, task *RequestTask, options TaskRunnerOptions) *TaskResult
	// https://github.com/ewasm/wasm-metering
	UseRequestGas func(gas int) bool

	HandleTaskResult   func(*RequestTask, int, *TaskResult, chan *TaskResult, chan bool)
	HandleNoCacheFound func(*RequestTask, *TaskResult) error

	Retry      int
	RetryDelay time.Duration

	OnFail    func(task *TaskResult)
	OnSuccess func(task *TaskResult)
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

func (task *RequestTask) EnsureCustomData() map[string]interface{} {
	if task.CustomData == nil {
		task.CustomData = map[string]interface{}{}
	}
	return task.CustomData
}

func (task *RequestTask) GetCustomData(key string) interface{} {
	if task.CustomData == nil {
		return nil
	}
	return task.CustomData[key]
}

func (task *RequestTask) SetCustomData(key string, data interface{}) {
	customData := task.EnsureCustomData()
	customData[key] = data
}

type TaskResult struct {
	Task          *RequestTask
	RequestResult *RequestResult
	Errors        []error
	FlowError     error
	IsVirtual     bool
	IsFromCache   bool
	HasDownloaded bool

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

func (reqSender *RequestSender) SendRequestWithTimeout(ctx context.Context, config *RequestConfig, expiryTime time.Duration) (*RequestResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	c, cancel := context.WithCancel(ctx)

	errChan := make(chan error, 0)
	resultChan := make(chan *RequestResult, 0)
	go func() {
		res, err := reqSender.client.SendRequest(c, config)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- res
		}
	}()

	select {
	case err := <-errChan:
		return nil, err
	case result := <-resultChan:
		return result, nil
	case <-time.After(expiryTime):
		cancel()
		return nil, errors.New("timeout")
	}
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
	if task.RequestConfig == nil {
		result.Errors = []error{errors.New("request config is empty(task: " + task.Name + ")")}
		return result
	} else if task.RequestConfig.Url == "" {
		result.Errors = []error{errors.New("url is empty(task: " + task.Name + ")")}
		return result
	}

	if options.UseRequestGas != nil && !options.UseRequestGas(1) {
		result.Errors = []error{errors.New("no gas left")}
		return result
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
		getDelayTime = func(task *RequestTask, taskIndex int, options *TaskRunnerOptions, result *TaskResult) time.Duration {
			if result != nil && (result.IsFromCache || result.IsVirtual) {
				return 0
			}
			return task.TaskRunnerOptions.RequestInterval
		}
	}

	tryEarlyDone := options.TryEarlyDone
	if tryEarlyDone == nil {
		tryEarlyDone = func(task *RequestTask, taskIndex int, options *TaskRunnerOptions, result *TaskResult) bool {
			if result != nil && (result.IsFromCache || result.IsVirtual) {
				return true
			}
			return false
		}
	}

	getCacheOrVirtual := options.GetCacheOrVirtual
	if getCacheOrVirtual == nil {
		getCacheOrVirtual = func(ctx context.Context, task *RequestTask, options TaskRunnerOptions) *TaskResult { return nil }
	}

	handleTaskResult := options.HandleTaskResult
	if handleTaskResult == nil {
		handleTaskResult = func(task *RequestTask, taskIndex int, taskResult *TaskResult, taskResultChan chan *TaskResult, cancelChan chan bool) {
			taskResultChan <- taskResult
		}
	}

	handleNoCacheFound := options.HandleNoCacheFound
	if handleNoCacheFound == nil {
		handleNoCacheFound = func(task *RequestTask, taskResult *TaskResult) error {
			return nil
		}
	}

	stop := func() {
		stopped = true
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
			run := func() *TaskResult {
				task := t
				defer wg.Done()
				result := getCacheOrVirtual(nil, task, options)
				if result == nil {
					err := handleNoCacheFound(task, result)
					if err != nil {
						result = &TaskResult{
							Task:      task,
							FlowError: err,
						}
						handleTaskResult(task, i, result, taskResultChan, cancelChan)
						stop()
						return result
					}
					result = reqSender.RunTask(nil, task, options)
					if task.TransformFromResponse != nil && result.RequestResult != nil {
						task.TransformFromResponse(task, i, result)
					}
					result.HasDownloaded = true
					result.IsFromCache = false
				} else {
					if !result.IsVirtual {
						if task.TransformFromCache != nil {
							task.TransformFromCache(task, i, result)
						}
						result.IsFromCache = true
					}
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
				return result
			}
			earlyDoneChan := make(chan bool, 0)
			var result *TaskResult
			if options.WaitForPreviousTask {
				wg.Add(1)
				result = run()
			} else {
				wg.Add(1)
				go func() {
					result := run()
					if tryEarlyDone(t, i, &options, result) {
						earlyDoneChan <- true
					}
				}()
			}
			if i == len(tasks)-1 {
				break
			}
			if !stopped {
				select {
				case <-earlyDoneChan:
				case <-time.After(getDelayTime(t, i, &options, result)):
				}
			}
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

var BasicRequestSenderInst *RequestSender = NewRequestSender([]string{})
