package processor

import (
	"context"
	"time"

	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-server/jobsdb"
	"github.com/rudderlabs/rudder-server/services/rsources"
)

// workerHandle is the interface trying to abstract processor's [Handle] implememtation from the worker
type workerHandle interface {
	logger() logger.Logger
	config() workerHandleConfig
	rsourcesService() rsources.JobService
	handlePendingGatewayJobs(key string) bool
	stats() *processorStats
	parseTJobs(job jobsdb.JobsResult) *transformationMessage

	getGWJobs(partition string) jobsdb.JobsResult
	markExecuting(jobs []*jobsdb.JobT, db jobsdb.JobsDB) error
	jobSplitter(jobs []*jobsdb.JobT, rsourcesStats rsources.StatsCollector) []subJob
	processJobsForDest(partition string, subJobs subJob) *transformationMessage
	multiplex(partition string, subJobs subJob) *tStore
	storeToTransformDB(ctx context.Context, partition string, storeJob *tStore) error
	getTransformJobs(ctx context.Context, partition string) (j jobsdb.JobsResult)
	transformations(partition string, in *transformationMessage) *storeMessage
	Store(ctx context.Context, partition string, in *storeMessage)
}

// workerHandleConfig is a struct containing the processor.Handle configuration relevant for workers
type workerHandleConfig struct {
	maxEventsToProcess int

	enablePipelining      bool
	pipelineBufferedItems int
	subJobSize            int

	readLoopSleep time.Duration
	maxLoopSleep  time.Duration

	gwDB        jobsdb.JobsDB
	transformDB jobsdb.JobsDB
}
