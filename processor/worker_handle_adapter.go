package processor

import (
	"time"

	"github.com/rudderlabs/rudder-go-kit/logger"
	backendconfig "github.com/rudderlabs/rudder-server/backend-config"
	"github.com/rudderlabs/rudder-server/jobsdb"
	"github.com/rudderlabs/rudder-server/processor/transformer"
	"github.com/rudderlabs/rudder-server/services/rsources"
	"github.com/rudderlabs/rudder-server/utils/misc"
	"github.com/rudderlabs/rudder-server/utils/types"
	"github.com/samber/lo"
)

// workerHandleAdapter is a wrapper around processor.Handle that implements the workerHandle interface
type workerHandleAdapter struct {
	*Handle
}

func (h *workerHandleAdapter) logger() logger.Logger {
	return h.Handle.logger
}

func (h *workerHandleAdapter) config() workerHandleConfig {
	return workerHandleConfig{
		enablePipelining:      h.Handle.config.enablePipelining,
		pipelineBufferedItems: h.Handle.config.pipelineBufferedItems,
		maxEventsToProcess:    h.Handle.config.maxEventsToProcess,
		subJobSize:            h.Handle.config.subJobSize,
		readLoopSleep:         h.Handle.config.readLoopSleep,
		maxLoopSleep:          h.Handle.config.maxLoopSleep,
		gwDB:                  h.Handle.gatewayDB,
		transformDB:           h.Handle.transformDB,
	}
}

func (h *workerHandleAdapter) rsourcesService() rsources.JobService {
	return h.Handle.rsourcesService
}

func (h *workerHandleAdapter) stats() *processorStats {
	return &h.Handle.stats
}

func (h *workerHandleAdapter) parseTJobs(job jobsdb.JobsResult) *transformationMessage {
	jobList := job.Jobs
	statusList := make([]*jobsdb.JobStatusT, 0)
	groupedEvents := make(map[string][]transformer.TransformerEvent) // grouped by srcDstKey
	trackingPlanEnabledMap := make(map[SourceIDT]bool)
	eventsByMessageID := make(map[string]types.SingularEventWithReceivedAt)

	for _, j := range jobList {
		var (
			params ParametersT
			event  types.SingularEventT
		)
		_ = jsonfast.Unmarshal(j.Parameters, &params)
		_ = jsonfast.Unmarshal(j.EventPayload, &event)
		sourceId := params.SourceID
		receivedAt := lo.Must(time.Parse(misc.RFC3339Milli, params.ReceivedAt))
		trackingPlanEnabledMap[SourceIDT(sourceId)] = params.TrackingPlanEnabled
		source, sourceError := h.getSourceBySourceID(sourceId)
		if sourceError != nil {
			h.logger().Errorf("Dropping Job since Source not found for sourceId %q: %v", sourceId, sourceError)
			// append aborted status to the job
			// TODO
			statusList = append(statusList, &jobsdb.JobStatusT{
				JobID:         j.JobID,
				JobState:      jobsdb.Aborted.State,
				AttemptNum:    1,
				ExecTime:      time.Now(),
				RetryTime:     time.Now(),
				ErrorCode:     "0",
				ErrorResponse: []byte(`{"success":"source not found"}`),
				Parameters:    []byte(`{}`),
				JobParameters: j.Parameters,
				WorkspaceId:   j.WorkspaceId,
			})
			continue
		}
		destId := params.DestinationID
		srcDstKey := getKeyFromSourceAndDest(sourceId, destId)
		eventsByMessageID[params.MessageID] = types.SingularEventWithReceivedAt{
			SingularEvent: event,
			ReceivedAt:    receivedAt,
		}
		commonMetadataFromSingularEvent := makeCommonMetadataFromSingularEvent(
			event,
			j,
			receivedAt,
			source,
			types.EventParams{
				SourceId:        sourceId,
				SourceJobRunId:  params.SourceJobRunID,
				SourceTaskRunId: params.SourceTaskRunID,
			},
		)
		shallowEventCopy := transformer.TransformerEvent{}
		shallowEventCopy.Message = event
		enhanceWithTimeFields(&shallowEventCopy, event, receivedAt)
		enhanceWithMetadata(
			commonMetadataFromSingularEvent,
			&shallowEventCopy,
			&backendconfig.DestinationT{},
		)

		// TODO: TP ID preference 1.event.context set by rudderTyper   2.From WorkSpaceConfig (currently being used)
		shallowEventCopy.Metadata.TrackingPlanId = source.DgSourceTrackingPlanConfig.TrackingPlan.Id
		shallowEventCopy.Metadata.TrackingPlanVersion = source.DgSourceTrackingPlanConfig.TrackingPlan.Version
		shallowEventCopy.Metadata.SourceTpConfig = source.DgSourceTrackingPlanConfig.Config
		shallowEventCopy.Metadata.MergedTpConfig = source.DgSourceTrackingPlanConfig.GetMergedConfig(commonMetadataFromSingularEvent.EventType)

		groupedEvents[srcDstKey] = append(groupedEvents[srcDstKey], shallowEventCopy)
		statusList = append(statusList, &jobsdb.JobStatusT{
			JobID:         j.JobID,
			JobState:      jobsdb.Succeeded.State,
			AttemptNum:    1,
			ExecTime:      time.Now(),
			RetryTime:     time.Now(),
			ErrorCode:     "200",
			ErrorResponse: []byte(`{"success":"OK"}`),
			Parameters:    []byte(`{}`),
			JobParameters: j.Parameters,
			WorkspaceId:   j.WorkspaceId,
		})
	}

	return &transformationMessage{
		groupedEvents:          groupedEvents,
		trackingPlanEnabledMap: trackingPlanEnabledMap,
		eventsByMessageID:      eventsByMessageID,
		statusList:             statusList,
	}
}
