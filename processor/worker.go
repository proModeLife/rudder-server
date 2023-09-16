package processor

import (
	"context"
	"sync"
	"time"

	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-server/rruntime"
	"github.com/rudderlabs/rudder-server/services/rsources"
	"github.com/rudderlabs/rudder-server/utils/misc"
)

// newMultiplexWorker creates a new worker
func newMultiplexWorker(partition string, h workerHandle) *worker {
	w := &worker{
		handle:    h,
		logger:    h.logger().Child(partition),
		partition: partition,
	}
	w.lifecycle.ctx, w.lifecycle.cancel = context.WithCancel(context.Background())
	w.channel.preprocess = make(chan subJob, w.handle.config().pipelineBufferedItems)
	w.channel.store = make(chan *tStore, (w.handle.config().pipelineBufferedItems+1)*(w.handle.config().maxEventsToProcess/w.handle.config().subJobSize+1))
	w.start()

	return w
}

// worker picks jobs from jobsdb for the appropriate partition and performs all processing steps for them
//  1. preprocess
//  2. transform
//  3. store
type worker struct {
	partition string
	handle    workerHandle
	logger    logger.Logger

	lifecycle struct { // worker lifecycle related fields
		ctx    context.Context    // worker context
		cancel context.CancelFunc // worker context cancel function
		wg     sync.WaitGroup     // worker wait group
	}
	channel struct { // worker channels
		preprocess chan subJob  // preprocess channel is used to send jobs to preprocess asynchronously when pipelining is enabled
		store      chan *tStore // store channel is used to send jobs to store asynchronously when pipelining is enabled
	}
}

// start starts the various worker goroutines
func (w *worker) start() {
	if !w.handle.config().enablePipelining {
		return
	}

	// wait for context to be cancelled
	w.lifecycle.wg.Add(1)
	rruntime.Go(func() {
		defer w.lifecycle.wg.Done()
		defer close(w.channel.preprocess)
		<-w.lifecycle.ctx.Done()
	})

	// preprocess jobs
	w.lifecycle.wg.Add(1)
	rruntime.Go(func() {
		defer w.lifecycle.wg.Done()
		defer close(w.channel.store)
		defer w.logger.Debugf("preprocessing routine stopped for worker: %s", w.partition)
		for jobs := range w.channel.preprocess {
			w.channel.store <- w.handle.multiplex(w.partition, jobs)
		}
	})

	// store jobs
	w.lifecycle.wg.Add(1)
	rruntime.Go(func() {
		defer w.lifecycle.wg.Done()
		defer w.logger.Debugf("store routine stopped for worker: %s", w.partition)
		for tStoreJob := range w.channel.store {
			if err := w.handle.storeToTransformDB(w.lifecycle.ctx, w.partition, tStoreJob); err != nil {
				panic(err)
			}
		}
	})
}

// Work picks the next set of jobs from the jobsdb and returns [true] if jobs were picked, [false] otherwise
func (w *worker) Work() (worked bool) {
	if !w.handle.config().enablePipelining {
		return w.handle.handlePendingGatewayJobs(w.partition)
	}
	start := time.Now()
	jobs := w.handle.getGWJobs(w.partition)
	afterGetJobs := time.Now()
	if len(jobs.Jobs) == 0 {
		return
	}
	worked = true

	if err := w.handle.markExecuting(jobs.Jobs, w.handle.config().gwDB); err != nil {
		w.logger.Error(err)
		panic(err)
	}

	w.handle.stats().DBReadThroughput.Count(throughputPerSecond(jobs.EventsCount, time.Since(start)))

	rsourcesStats := rsources.NewStatsCollector(w.handle.rsourcesService())
	rsourcesStats.BeginProcessing(jobs.Jobs)
	w.channel.preprocess <- subJob{
		subJobs:       jobs.Jobs,
		rsourcesStats: rsourcesStats,
	}

	if !jobs.LimitsReached {
		readLoopSleep := w.handle.config().readLoopSleep
		if elapsed := time.Since(afterGetJobs); elapsed < readLoopSleep {
			if err := misc.SleepCtx(w.lifecycle.ctx, readLoopSleep-elapsed); err != nil {
				return
			}
		}
	}

	return
}

func (w *worker) SleepDurations() (min, max time.Duration) {
	return w.handle.config().readLoopSleep, w.handle.config().maxLoopSleep
}

// Stop stops the worker and waits until all its goroutines have stopped
func (w *worker) Stop() {
	w.lifecycle.cancel()
	w.lifecycle.wg.Wait()
}
