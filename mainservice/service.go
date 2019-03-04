package mainservice

import (
	// "context"
	"fmt"
	"github.com/xtforgame/agak/agapiserver"
	"github.com/xtforgame/agak/config"
	"github.com/xtforgame/agak/requestsender"
	"github.com/xtforgame/agak/scheduler"
	"github.com/xtforgame/agak/utils"
	// "sort"
	// "strings"
	// "runtime"
	"sync"
	"time"
	// "encoding/json"
	// "io/ioutil"
	// "os"
	// "strings"
)

type SbMainServiceOptions struct {
}

type SbMainService struct {
	config        *config.Config
	options       SbMainServiceOptions
	mainScheduler *scheduler.Scheduler
	httpServer    *agapiserver.HttpServer
	reqSender     *requestsender.RequestSender
}

func NewSbMainService(c *config.Config, options SbMainServiceOptions) *SbMainService {
	fmt.Println("\n=======================")
	fmt.Println("config :", c)
	fmt.Println("=======================\n")

	ms := &SbMainService{
		config:        c,
		options:       options,
		mainScheduler: scheduler.NewScheduler(utils.LocTaipei),
		httpServer:    agapiserver.NewHttpServer(),
		reqSender:     requestsender.NewRequestSender(c.RequestSender.Proxies),
	}
	return ms
}

// ===============

func (ms *SbMainService) GetConfig() *config.Config {
	return ms.config
}

func (ms *SbMainService) GetOptions() SbMainServiceOptions {
	return ms.options
}

func (ms *SbMainService) GetMainScheduler() *scheduler.Scheduler {
	return ms.mainScheduler
}

func (ms *SbMainService) GetHttpServer() *agapiserver.HttpServer {
	return ms.httpServer
}

func (ms *SbMainService) GetReqSender() *requestsender.RequestSender {
	return ms.reqSender
}

// ===============

func (ms *SbMainService) Init() {
	ms.mainScheduler.Init()
	ms.httpServer.Init()
}

func (ms *SbMainService) Destroy() {
	ms.mainScheduler.Destroy()
}

type NewScheduleOptions struct {
	FirstRunStartAt  *time.Time
	FirstRunDeadline *time.Time
}

func (ms *SbMainService) newSchedule0(
	jobName string,
	spec string,
	runFunc func(*scheduler.Job, *scheduler.Entry),
) (*scheduler.Entry, error) {
	job := &scheduler.Job{Name: jobName}
	entry, err := ms.mainScheduler.AddEntry(spec, job, func(ent *scheduler.Entry) {
		job.RunFunc = func() {
			e := ent
			runFunc(e.GetJob(), e)
		}
	})

	return entry, err
}

func (ms *SbMainService) newSchedule(
	jobName string,
	spec string,
	runFunc func(*scheduler.Job, *scheduler.Entry),
	options NewScheduleOptions,
) (*scheduler.Entry, error) {
	var mu sync.Mutex
	firstRunDone := false
	race := func() bool {
		mu.Lock()
		shouldRun := !firstRunDone
		firstRunDone = true
		defer mu.Unlock()
		return shouldRun
	}
	runOnce := false
	dailySaveEntry, err := ms.newSchedule0(
		jobName,
		spec,
		func(j *scheduler.Job, e *scheduler.Entry) {
			if runOnce || race() {
				runFunc(j, e)
			}
			runOnce = true
		},
	)
	if err != nil {
		return nil, err
	}
	dailySaveJob := dailySaveEntry.GetJob()
	if options.FirstRunStartAt != nil && options.FirstRunDeadline != nil {
		nowNs := utils.TwNow().UnixNano()
		startAtNs := options.FirstRunStartAt.UnixNano()
		deadlineNs := options.FirstRunDeadline.UnixNano()
		if nowNs >= startAtNs && nowNs < deadlineNs {
			go func() {
				if race() {
					runFunc(dailySaveJob, dailySaveEntry)
					time.Sleep(time.Second)
					runOnce = true
				}
			}()
		}
	}
	return dailySaveEntry, err
}

type DailyScheduleTime struct {
	Hour   int
	Minute int
	Second int
}

func (ds DailyScheduleTime) GetDurationText() string {
	return fmt.Sprintf("%02dh%02dm%02ds", ds.Hour, ds.Minute, ds.Second)
}

func (ds DailyScheduleTime) GetScheduleTimeText() string {
	return fmt.Sprintf("%d %d %d * * *", ds.Second, ds.Minute, ds.Hour)
}

func (ms *SbMainService) NewDailySchedule(
	name string,
	start DailyScheduleTime,
	deadline DailyScheduleTime,
	runFunc func(job *scheduler.Job, entry *scheduler.Entry),
) (*scheduler.Entry, error) {
	startOfDay := utils.TwStartOfDay(utils.TwNow())
	firstRunStartAt, _ := utils.AddDuration(startOfDay, start.GetDurationText())
	firstRunDeadline, _ := utils.AddDuration(startOfDay, deadline.GetDurationText())
	entry, err := ms.newSchedule(
		name,
		start.GetScheduleTimeText(),
		runFunc,
		NewScheduleOptions{
			FirstRunStartAt:  firstRunStartAt,
			FirstRunDeadline: firstRunDeadline,
		},
	)
	if err != nil {
		panic("failed to add " + name + " job")
	}
	fmt.Println(name+" Entry :", entry)
	return entry, err
}

func (ms *SbMainService) dailyGreeting() (*scheduler.Entry, error) {
	dailySaveJob := &scheduler.Job{Name: "Daily Greeting"}
	dailySaveJob.RunFunc = func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			fmt.Println("Hello world")
			// res, err := ms.reqSender.SendRequest(
			// 	nil,
			// 	&requestsender.RequestConfig{
			// 		Method: "POST",
			// 		Url:    "https://httpbin.org/post",
			// 		Header: map[string]string{
			// 			"Content-Type": "application/json",
			// 		},
			// 		Body:      []byte(`["3338"]`),
			// 		Validator: requestsender.DefaultResponseValidator,
			// 	},
			// )
			// if err == nil {
			// 	fmt.Println("res", string(res.Body))
			// }
			wg.Done()
		}()
		wg.Wait()
	}
	// return ms.mainScheduler.AddEntry("0 * * * * *", dailySaveJob)
	return ms.mainScheduler.AddEntry("0 0 8 * * *", dailySaveJob, nil)
}

func (ms *SbMainService) RegisterAllJobs() {
	dailyGreetingEntry, _ := ms.dailyGreeting()

	// ms.mainScheduler.WaitForFinish()

	utils.HandleSIGINTandSIGTERM()

	ms.mainScheduler.RemoveEntry(dailyGreetingEntry)
	ms.mainScheduler.Stop()
}

func (ms *SbMainService) Start() {
	// utils.SlackAlert(ms.config.Slack.Webhook, "Start")
	// b, err := utils.ZipLocalFolder(stockfetcher.CacheFolder, "/twse-ticks")
	// if err == nil {
	// 	ioutil.WriteFile("xxxxxxx.zip", b, 0644)
	// 	err = utils.RemoveContents(stockfetcher.CacheFolder)
	// 	retryCounter := 0
	// 	for err != nil && retryCounter < 10 {
	// 		retryCounter++
	// 		err = utils.RemoveContents(stockfetcher.CacheFolder)
	// 	}
	// }
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		ms.RegisterAllJobs()

		// ms.tryUpgrade()
		// ms.tryRemoveLegacyData()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ms.httpServer.Start()
	}()
	// if simulationMode {
	// 	ms.liveTradeMain(&wg)
	// }
	wg.Wait()
}

func (ms *SbMainService) SchedulerWaitForFinishTest() {
	var entry0 *scheduler.Entry
	j0 := &scheduler.Job{
		Name: "Every(3)",
	}
	j0.RunFunc = func() {
		ms.mainScheduler.RemoveEntry(entry0)
		now0 := time.Now()
		fmt.Println(j0.Name, now0)
	}

	entry0, _ = ms.mainScheduler.AddEntry(
		"@every 3s",
		j0,
		nil,
	)
	fmt.Println("entry0 :", entry0)

	var entry1 *scheduler.Entry
	j1 := &scheduler.Job{
		Name: "Every(10)",
	}
	j1.RunFunc = func() {
		ms.mainScheduler.RemoveEntry(entry1)
		now1 := time.Now()
		fmt.Println(j1.Name, now1)
		ms.mainScheduler.Stop()
	}

	entry1, _ = ms.mainScheduler.AddEntry(
		"@every 10s",
		j1,
		nil,
	)

	// j.Remove()
	ms.mainScheduler.WaitForFinish()
}

func (ms *SbMainService) ScheduledRequestTest(proxy string) {
	counter := 0
	var entry0 *scheduler.Entry
	j0 := &scheduler.Job{
		Name: "Every(1)",
	}
	now0 := time.Now()
	fmt.Println("now0 :", now0)
	j0.RunFunc = func() {
		ms.mainScheduler.RemoveEntry(entry0)
		counter++
		if counter > 10 {
			return
		}
		entry0, _ = ms.mainScheduler.AddEntry(
			"@every 1s",
			j0,
			nil,
		)
		now0 := time.Now()
		go func() {
			c2 := counter
			time.Sleep(2)
			fmt.Println("now0 :", now0)
			if c2 == 10 {
				ms.mainScheduler.Stop()
			}
			// fmt.Println("r.Body :", string(r.Body))
		}()
	}

	entry0, _ = ms.mainScheduler.AddEntry(
		"@every 1s",
		j0,
		nil,
	)

	// j.Remove()
	ms.mainScheduler.WaitForFinish()
}
