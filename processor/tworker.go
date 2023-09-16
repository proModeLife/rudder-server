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

type tworker struct {
	partition string
	handle    workerHandle
	logger    logger.Logger

	lifecycle struct { // worker lifecycle related fields
		ctx    context.Context    // worker context
		cancel context.CancelFunc // worker context cancel function
		wg     sync.WaitGroup     // worker wait group
	}

	channel struct { // worker channels
		transform chan *transformationMessage // transform channel is used to send jobs to transform asynchronously when pipelining is enabled
		store     chan *storeMessage          // store channel is used to send jobs to store asynchronously when pipelining is enabled
	}
}

func newTransformWorker(partition string, h workerHandle) *tworker {
	w := &tworker{
		handle:    h,
		logger:    h.logger().Child(partition),
		partition: partition,
	}
	w.lifecycle.ctx, w.lifecycle.cancel = context.WithCancel(context.Background())
	w.channel.transform = make(chan *transformationMessage, w.handle.config().pipelineBufferedItems)
	w.channel.store = make(chan *storeMessage, (w.handle.config().pipelineBufferedItems+1)*(w.handle.config().maxEventsToProcess/w.handle.config().subJobSize+1))
	w.start()

	return w
}

// start starts the various worker goroutines
func (w *tworker) start() {
	if !w.handle.config().enablePipelining {
		return
	}

	// wait for context to be cancelled
	w.lifecycle.wg.Add(1)
	rruntime.Go(func() {
		defer w.lifecycle.wg.Done()
		defer close(w.channel.transform)
		<-w.lifecycle.ctx.Done()
	})

	// transform routine
	w.lifecycle.wg.Add(1)
	rruntime.Go(func() {
		defer w.lifecycle.wg.Done()
		defer close(w.channel.store)
		defer w.logger.Debugf("store routine stopped for tWorker: %s", w.partition)
		for tJob := range w.channel.transform {
			w.channel.store <- w.handle.transformations(w.partition, tJob)
		}
	})

	// store routine
	w.lifecycle.wg.Add(1)
	rruntime.Go(func() {
		defer w.lifecycle.wg.Done()
		defer w.logger.Debugf("store routine stopped for tWorker: %s", w.partition)
		for jobs := range w.channel.store {
			w.handle.Store(w.lifecycle.ctx, w.partition, jobs)
		}
	})
}

// stop stops the worker
func (w *tworker) Stop() {
	w.lifecycle.cancel()
	w.lifecycle.wg.Wait()
}

func (w *tworker) SleepDurations() (min, max time.Duration) {
	return w.handle.config().readLoopSleep, w.handle.config().maxLoopSleep
}

// work picks up jobs for the worker and returns whether it worked or not
func (w *tworker) Work() (worked bool) {
	if !w.handle.config().enablePipelining {
		return w.handle.handlePendingGatewayJobs(w.partition)
	}
	// start := time.Now() //-> used for some stats
	jobs := w.handle.getTransformJobs(w.lifecycle.ctx, w.partition)
	afterGetJobs := time.Now()
	if len(jobs.Jobs) == 0 {
		return
	}
	worked = true

	if err := w.handle.markExecuting(jobs.Jobs, w.handle.config().transformDB); err != nil {
		w.logger.Error(err)
		panic(err)
	}
	// add some stats here

	rsourcesStats := rsources.NewStatsCollector(w.handle.rsourcesService())
	rsourcesStats.BeginProcessing(jobs.Jobs)
	w.channel.transform <- w.handle.parseTJobs(jobs)

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
