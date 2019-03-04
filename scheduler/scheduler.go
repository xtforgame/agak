package scheduler

import (
	// "fmt"
	"github.com/robfig/cron/v3"
	"github.com/xtforgame/agak/utils"
	"sync"
	"time"
)

type EntryID = cron.EntryID

type Job struct {
	Name     string
	RunFunc  func()
	Metadata map[string]interface{}
}

func (job *Job) Run() {
	job.RunFunc()
}

// ====================

type Entry struct {
	scheduler *Scheduler
	entryId   cron.EntryID
	job       *Job
}

func (entry *Entry) GetJob() *Job {
	return entry.job
}

func (entry *Entry) GetEntryID() cron.EntryID {
	return entry.entryId
}

// func (entry *Entry) SetEntryID(entryId cron.EntryID) {
// 	entry.entryId = entryId
// }

// =============================
type Scheduler struct {
	cronScheduler *cron.Cron
	events        *utils.EventEmitter
	entries       []*Entry
	jobMux        sync.Mutex
	stopChan      chan bool
}

func NewScheduler(location *time.Location) *Scheduler {
	return &Scheduler{
		cronScheduler: cron.New(cron.WithLocation(location), cron.WithSeconds()),
		events:        utils.NewEventEmitter(),
		entries:       []*Entry{},
	}
}

func (scheduler *Scheduler) Init() {
	scheduler.Stop()
	scheduler.stopChan = make(chan bool, 0)
	scheduler.cronScheduler.Start()
	sc := make(chan interface{}, 0)
	scheduler.events.AddListener("stop", sc)
	scheduler.events.AddListener("done", sc)
	go func() {
		<-sc
		close(sc)
		scheduler.stopChan <- true
		close(scheduler.stopChan)
		scheduler.stopChan = nil
		scheduler.events.RemoveListener("stop", sc)
		scheduler.events.RemoveListener("done", sc)
	}()
}

func (scheduler *Scheduler) Stop() {
	if scheduler.stopChan != nil {
		scheduler.events.Emit("stop", struct{}{})
	}
}

func (scheduler *Scheduler) WaitForFinish() {
	if scheduler.stopChan != nil {
		sc := make(chan interface{}, 0)
		scheduler.events.AddListener("stop", sc)
		scheduler.events.AddListener("done", sc)
		<-sc
		close(sc)
		scheduler.events.RemoveListener("stop", sc)
		scheduler.events.RemoveListener("done", sc)
	}
}

func (scheduler *Scheduler) Destroy() {
	scheduler.Stop()
}

func (scheduler *Scheduler) AddEntry(
	spec string,
	job *Job,
	callback func(entry *Entry),
) (*Entry, error) {
	entry := &Entry{
		job: job,
	}
	if callback != nil {
		callback(entry)
	}

	var err error
	entry.entryId, err = scheduler.cronScheduler.AddJob(spec, job)
	if err != nil {
		return nil, err
	}

	scheduler.jobMux.Lock()
	scheduler.entries = append(scheduler.entries, entry)
	scheduler.jobMux.Unlock()
	scheduler.events.Emit("add-entry", entry)
	return entry, nil
}

func (scheduler *Scheduler) RemoveEntry(entry *Entry) {
	entryIsMine := false
	scheduler.jobMux.Lock()
	for i := range scheduler.entries {
		if scheduler.entries[i] == entry {
			entryIsMine = true
			break
		}
	}
	scheduler.jobMux.Unlock()
	if entryIsMine {
		scheduler.events.Emit("remove-entry", entry)
		scheduler.cronScheduler.Remove(entry.entryId)
	}
}

// =========================================================

func (scheduler *Scheduler) GetGcScheduler() *cron.Cron {
	return scheduler.cronScheduler
}
