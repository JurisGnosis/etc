package jobs

import (
	"log"
	"log/slog"
	"sync"
	"time"
)

var ticker *time.Ticker
var myJobs []Job
var myLock sync.Mutex
var selfStarted bool
var mapRunningJobs = map[string]bool{}

// 需要幂等
type Job interface {
	Execute() (done bool)
	Identifier() string
}

func init() {
	myLock = sync.Mutex{}
}

// execute only once
func Serve(PollingSeconds int) {
	myLock.Lock()
	if selfStarted {
		log.Fatalln("ticker already started!!!")
		return
	}
	log.Printf("start ticker every %d seconds\n", PollingSeconds)
	ticker = time.NewTicker(time.Duration(PollingSeconds) * time.Second)
	selfStarted = true
	myLock.Unlock()
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			myLock.Lock()
			var newJobs []Job
			for i := 0; i < len(myJobs); i++ {
				if !myJobs[i].Execute() {
					newJobs = append(newJobs, myJobs[i])
				} else {
					mapRunningJobs[myJobs[i].Identifier()] = false
				}
			}
			myJobs = newJobs
			myLock.Unlock()
		}
	}
}

// type Job should have Execute() method, returns (done bool).
// it will be executed every X secs until done is true
func AddJob(job Job) {
	myLock.Lock()
	if mapRunningJobs[job.Identifier()] {
		slog.Warn("job already running", "job", job.Identifier())
	} else {
		mapRunningJobs[job.Identifier()] = true
		myJobs = append(myJobs, job)
	}
	myLock.Unlock()
}
