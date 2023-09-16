package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rudderlabs/rudder-go-kit/config"
	"github.com/rudderlabs/rudder-go-kit/ro"
	backendconfig "github.com/rudderlabs/rudder-server/backend-config"
	"github.com/rudderlabs/rudder-server/jobsdb"
	"github.com/rudderlabs/rudder-server/processor/integrations"
	"github.com/rudderlabs/rudder-server/processor/transformer"
	"github.com/rudderlabs/rudder-server/services/dedup"
	"github.com/rudderlabs/rudder-server/services/rsources"
	"github.com/rudderlabs/rudder-server/utils/misc"
	"github.com/rudderlabs/rudder-server/utils/types"
	"golang.org/x/sync/errgroup"
)

// 1. takes jobs from gw jobsdb
//
// 2. multiplexes them * numConnections
//
// 3. stores to transform jobsdb
func (proc *Handle) multiplex(partition string, subJobs subJob) *tStore {
	if proc.limiter.preprocess != nil {
		defer proc.limiter.preprocess.BeginWithPriority(partition, proc.getLimiterPriority(partition))()
	}
	jobList := subJobs.subJobs
	start := time.Now()

	proc.stats.statNumRequests.Count(len(jobList))

	var statusList []*jobsdb.JobStatusT
	groupedEventsBySourceId := make(map[SourceIDT][]transformer.TransformerEvent)
	eventsByMessageID := make(map[string]types.SingularEventWithReceivedAt)
	var procErrorJobs []*jobsdb.JobT
	eventSchemaJobs := make([]*jobsdb.JobT, 0)
	archivalJobs := make([]*jobsdb.JobT, 0)

	totalEvents := 0

	proc.logger.Debug("[Processor] Total jobs picked up : ", len(jobList))

	marshalStart := time.Now()
	dedupKeys := make(map[string]struct{})
	sourceDupStats := make(map[dupStatKey]int)

	reportMetrics := make([]*types.PUReportedMetric, 0)
	inCountMap := make(map[string]int64)
	inCountMetadataMap := make(map[string]MetricMetadata)
	connectionDetailsMap := make(map[string]*types.ConnectionDetails)
	statusDetailsMap := make(map[string]map[string]*types.StatusDetail)

	outCountMap := make(map[string]int64) // destinations enabled
	destFilterStatusDetailMap := make(map[string]map[string]*types.StatusDetail)

	for _, batchEvent := range jobList {

		var gatewayBatchEvent types.GatewayBatchRequest
		err := jsonfast.Unmarshal(batchEvent.EventPayload, &gatewayBatchEvent)
		if err != nil {
			proc.logger.Warnf("json parsing of event payload for %s: %v", batchEvent.JobID, err)
			gatewayBatchEvent.Batch = []types.SingularEventT{}
		}
		var eventParams types.EventParams
		err = jsonfast.Unmarshal(batchEvent.Parameters, &eventParams)
		if err != nil {
			panic(err)
		}
		sourceId := eventParams.SourceId
		requestIP := gatewayBatchEvent.RequestIP
		receivedAt := gatewayBatchEvent.ReceivedAt

		// Iterate through all the events in the batch
		for _, singularEvent := range gatewayBatchEvent.Batch {
			messageId := misc.GetStringifiedData(singularEvent["messageId"])
			source, sourceError := proc.getSourceBySourceID(sourceId)
			if sourceError != nil {
				proc.logger.Errorf("Dropping Job since Source not found for sourceId %q: %v", sourceId, sourceError)
				continue
			}

			if proc.config.enableDedup {
				payload, _ := jsonfast.Marshal(singularEvent)
				messageSize := int64(len(payload))
				dedupKey := fmt.Sprintf("%v%v", messageId, eventParams.SourceJobRunId)
				if ok, previousSize := proc.dedup.Set(dedup.KeyValue{Key: dedupKey, Value: messageSize}); !ok {
					proc.logger.Debugf("Dropping event with duplicate dedupKey: %s", dedupKey)
					sourceDupStats[dupStatKey{sourceID: source.ID, equalSize: messageSize == previousSize}] += 1
					continue
				}
				dedupKeys[dedupKey] = struct{}{}
			}

			proc.updateSourceEventStatsDetailed(singularEvent, sourceId)

			// We count this as one, not destination specific ones
			totalEvents++
			eventsByMessageID[messageId] = types.SingularEventWithReceivedAt{
				SingularEvent: singularEvent,
				ReceivedAt:    receivedAt,
			}

			commonMetadataFromSingularEvent := makeCommonMetadataFromSingularEvent(
				singularEvent,
				batchEvent,
				receivedAt,
				source,
				eventParams,
			)

			payloadFunc := ro.Memoize(func() json.RawMessage {
				if proc.transientSources.Apply(source.ID) {
					return nil
				}
				payloadBytes, err := jsonfast.Marshal(singularEvent)
				if err != nil {
					return nil
				}
				return payloadBytes
			},
			)
			if proc.config.eventSchemaV2Enabled && // schemas enabled
				// source has schemas enabled or if we override schemas for all sources
				(source.EventSchemasEnabled || proc.config.eventSchemaV2AllSources) &&
				// TODO: could use source.SourceDefinition.Category instead?
				commonMetadataFromSingularEvent.SourceJobRunID == "" {
				if payload := payloadFunc(); payload != nil {
					eventSchemaJobs = append(eventSchemaJobs,
						&jobsdb.JobT{
							UUID:         batchEvent.UUID,
							UserID:       batchEvent.UserID,
							Parameters:   batchEvent.Parameters,
							CustomVal:    batchEvent.CustomVal,
							EventPayload: payload,
							CreatedAt:    time.Now(),
							ExpireAt:     time.Now(),
							WorkspaceId:  batchEvent.WorkspaceId,
						},
					)
				}
			}

			if proc.config.archivalEnabled &&
				commonMetadataFromSingularEvent.SourceJobRunID == "" { // archival enabled
				if payload := payloadFunc(); payload != nil {
					archivalJobs = append(archivalJobs,
						&jobsdb.JobT{
							UUID:         batchEvent.UUID,
							UserID:       batchEvent.UserID,
							Parameters:   batchEvent.Parameters,
							CustomVal:    batchEvent.CustomVal,
							EventPayload: payload,
							CreatedAt:    time.Now(),
							ExpireAt:     time.Now(),
							WorkspaceId:  batchEvent.WorkspaceId,
						},
					)
				}
			}

			// REPORTING - GATEWAY metrics - START
			// dummy event for metrics purposes only
			event := &transformer.TransformerResponse{}
			if proc.isReportingEnabled() {
				event.Metadata = *commonMetadataFromSingularEvent
				proc.updateMetricMaps(
					inCountMetadataMap,
					inCountMap,
					connectionDetailsMap,
					statusDetailsMap,
					event,
					jobsdb.Succeeded.State,
					types.GATEWAY,
					func() json.RawMessage {
						if payload := payloadFunc(); payload != nil {
							return payload
						}
						return []byte("{}")
					},
				)
			}
			// REPORTING - GATEWAY metrics - END

			// Getting all the destinations which are enabled for this
			// event
			if !proc.isDestinationAvailable(singularEvent, sourceId) {
				continue
			}

			if _, ok := groupedEventsBySourceId[SourceIDT(sourceId)]; !ok {
				groupedEventsBySourceId[SourceIDT(sourceId)] = make([]transformer.TransformerEvent, 0)
			}
			shallowEventCopy := transformer.TransformerEvent{}
			shallowEventCopy.Message = singularEvent
			shallowEventCopy.Message["request_ip"] = requestIP
			enhanceWithTimeFields(&shallowEventCopy, singularEvent, receivedAt)
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

			groupedEventsBySourceId[SourceIDT(sourceId)] = append(groupedEventsBySourceId[SourceIDT(sourceId)], shallowEventCopy)

			if proc.isReportingEnabled() {
				proc.updateMetricMaps(inCountMetadataMap, outCountMap, connectionDetailsMap, destFilterStatusDetailMap, event, jobsdb.Succeeded.State, types.DESTINATION_FILTER, func() json.RawMessage { return []byte(`{}`) })
			}
		}

		// Mark the batch event as processed
		newStatus := jobsdb.JobStatusT{
			JobID:         batchEvent.JobID,
			JobState:      jobsdb.Succeeded.State,
			AttemptNum:    1,
			ExecTime:      time.Now(),
			RetryTime:     time.Now(),
			ErrorCode:     "200",
			ErrorResponse: []byte(`{"success":"OK"}`),
			Parameters:    []byte(`{}`),
			JobParameters: batchEvent.Parameters,
			WorkspaceId:   batchEvent.WorkspaceId,
		}
		statusList = append(statusList, &newStatus)
	}

	// REPORTING - GATEWAY metrics - START
	if proc.isReportingEnabled() {
		types.AssertSameKeys(connectionDetailsMap, statusDetailsMap)
		for k, cd := range connectionDetailsMap {
			for _, sd := range statusDetailsMap[k] {
				m := &types.PUReportedMetric{
					ConnectionDetails: *cd,
					PUDetails:         *types.CreatePUDetails("", types.GATEWAY, false, true),
					StatusDetail:      sd,
				}
				reportMetrics = append(reportMetrics, m)
			}

			for _, dsd := range destFilterStatusDetailMap[k] {
				destFilterMetric := &types.PUReportedMetric{
					ConnectionDetails: *cd,
					PUDetails:         *types.CreatePUDetails(types.GATEWAY, types.DESTINATION_FILTER, false, false),
					StatusDetail:      dsd,
				}
				reportMetrics = append(reportMetrics, destFilterMetric)
			}
		}
		// empty failedCountMap because no failures,
		// events are just dropped at this point if no destination is found to route the events
		diffMetrics := getDiffMetrics(
			types.GATEWAY,
			types.DESTINATION_FILTER,
			inCountMetadataMap,
			inCountMap,
			outCountMap,
			map[string]int64{},
		)
		reportMetrics = append(reportMetrics, diffMetrics...)
	}
	// REPORTING - GATEWAY metrics - END

	proc.stats.statNumEvents.Count(totalEvents)

	marshalTime := time.Since(marshalStart)
	defer proc.stats.marshalSingularEvents.SendTiming(marshalTime)

	// TRACKING PLAN - START
	// Placing the trackingPlan validation filters here.
	// Else further down events are duplicated by destId, so multiple validation takes places for same event
	validateEventsStart := time.Now()
	validatedEventsBySourceId, validatedReportMetrics, validatedErrorJobs, trackingPlanEnabledMap := proc.validateEvents(groupedEventsBySourceId, eventsByMessageID)
	validateEventsTime := time.Since(validateEventsStart)
	defer proc.stats.validateEventsTime.SendTiming(validateEventsTime)

	// Appending validatedErrorJobs to procErrorJobs
	procErrorJobs = append(procErrorJobs, validatedErrorJobs...)

	// Appending validatedReportMetrics to reportMetrics
	reportMetrics = append(reportMetrics, validatedReportMetrics...)
	// TRACKING PLAN - END

	jobsToTransform := make([]*jobsdb.JobT, 0)
	// The below part further segregates events by sourceID and DestinationID.
	for sourceIdT, eventList := range validatedEventsBySourceId {
		trackingPlanEnabled := trackingPlanEnabledMap[sourceIdT]
		for idx := range eventList {
			event := &eventList[idx]
			sourceId := string(sourceIdT)
			singularEvent := event.Message

			backendEnabledDestTypes := proc.getBackendEnabledDestinationTypes(sourceId)
			enabledDestTypes := integrations.FilterClientIntegrations(singularEvent, backendEnabledDestTypes)
			workspaceID := eventList[idx].Metadata.WorkspaceID

			for i := range enabledDestTypes {
				destType := &enabledDestTypes[i]
				enabledDestinationsList := proc.filterDestinations(
					singularEvent,
					proc.getEnabledDestinations(sourceId, *destType),
				)

				// Adding a singular event multiple times if there are multiple destinations of same type
				for idx := range enabledDestinationsList {
					destination := &enabledDestinationsList[idx]
					transformAt := "processor"
					if val, ok := destination.
						DestinationDefinition.Config["transformAtV1"].(string); ok {
						transformAt = val
					}
					// Check for overrides through env
					transformAtOverrideFound := config.IsSet("Processor." + destination.DestinationDefinition.Name + ".transformAt")
					if transformAtOverrideFound {
						transformAt = config.GetString("Processor."+destination.DestinationDefinition.Name+".transformAt", "processor")
					}
					params := ParametersT{
						SourceID:                sourceId,
						DestinationID:           destination.ID,
						ReceivedAt:              event.Metadata.ReceivedAt,
						TransformAt:             transformAt,
						MessageID:               event.Metadata.MessageID,
						GatewayJobID:            event.Metadata.JobID,
						SourceTaskRunID:         event.Metadata.SourceTaskRunID,
						SourceJobRunID:          event.Metadata.SourceJobRunID,
						SourceJobID:             event.Metadata.SourceJobID,
						EventName:               event.Metadata.EventName,
						EventType:               event.Metadata.EventType,
						SourceCategory:          event.Metadata.SourceCategory,
						SourceDefinitionID:      event.Metadata.SourceDefinitionID,
						DestinationDefinitionID: destination.DestinationDefinition.ID,
						RecordID:                event.Metadata.RecordID,
						WorkspaceId:             workspaceID,
						TrackingPlanEnabled:     trackingPlanEnabled,
					}
					marshalledParams, err := jsonfast.Marshal(params)
					if err != nil {
						proc.logger.Errorf("[Processor] Failed to marshal parameters object. Parameters: %v", params)
						panic(err)
					}

					payload, _ := jsonfast.Marshal(singularEvent)

					id := misc.FastUUID()
					rudderID := event.Metadata.RudderID
					if rudderID == "" {
						rudderID = "random-" + id.String()
					}

					jobsToTransform = append(jobsToTransform, &jobsdb.JobT{
						UUID:         id,
						UserID:       rudderID,
						Parameters:   marshalledParams,
						CreatedAt:    time.Now(),
						ExpireAt:     time.Now(),
						CustomVal:    *destType,
						WorkspaceId:  workspaceID,
						EventPayload: payload,
					})
				}
			}
		}
	}

	if len(statusList) != len(jobList) {
		panic(fmt.Errorf("len(statusList):%d != len(jobList):%d", len(statusList), len(jobList)))
	}
	processTime := time.Since(start)
	proc.stats.processJobsTime.SendTiming(processTime)
	processJobThroughput := throughputPerSecond(totalEvents, processTime)
	// processJob throughput per second.
	proc.stats.processJobThroughput.Count(processJobThroughput)

	return &tStore{
		tJobs:           jobsToTransform,
		procErrorJobs:   procErrorJobs,
		statusList:      statusList,
		eventSchemaJobs: eventSchemaJobs,
		archivalJobs:    archivalJobs,
		reports:         reportMetrics,
		rsourcesStats:   subJobs.rsourcesStats,
	}
}

type tStore struct {
	tJobs           []*jobsdb.JobT
	procErrorJobs   []*jobsdb.JobT
	archivalJobs    []*jobsdb.JobT
	eventSchemaJobs []*jobsdb.JobT
	reports         []*types.PUReportedMetric
	statusList      []*jobsdb.JobStatusT
	rsourcesStats   rsources.StatsCollector
	dedupKeys       map[string]struct{} // TODO
	sourceDupStatus map[dupStatKey]int  // TODO
}

func (proc *Handle) storeToTransformDB(ctx context.Context, partition string, storeJob *tStore) error {
	if proc.limiter.tStore != nil {
		defer proc.limiter.tStore.BeginWithPriority(partition, proc.getLimiterPriority(partition))()
	}
	g, groupCtx := errgroup.WithContext(ctx)

	eventSchemaJobs := storeJob.eventSchemaJobs
	archivalJobs := storeJob.archivalJobs
	jobsToTransform := storeJob.tJobs
	procErrorJobs := storeJob.procErrorJobs
	reportMetrics := storeJob.reports
	statusList := storeJob.statusList

	g.Go(func() error {
		if len(eventSchemaJobs) == 0 {
			return nil
		}
		err := misc.RetryWithNotify(
			groupCtx,
			proc.jobsDBCommandTimeout,
			proc.jobdDBMaxRetries,
			func(ctx context.Context) error {
				return proc.eventSchemaDB.WithStoreSafeTx(
					ctx,
					func(tx jobsdb.StoreSafeTx) error {
						return proc.eventSchemaDB.StoreInTx(ctx, tx, eventSchemaJobs)
					},
				)
			}, proc.sendRetryStoreStats)
		if err != nil {
			return fmt.Errorf("store into event schema table failed with error: %v", err)
		}
		proc.logger.Debug("[Processor] Total jobs written to event_schema: ", len(eventSchemaJobs))
		return nil
	})

	g.Go(func() error {
		if len(archivalJobs) == 0 {
			return nil
		}
		err := misc.RetryWithNotify(
			groupCtx,
			proc.jobsDBCommandTimeout,
			proc.jobdDBMaxRetries,
			func(ctx context.Context) error {
				return proc.archivalDB.WithStoreSafeTx(
					ctx,
					func(tx jobsdb.StoreSafeTx) error {
						return proc.archivalDB.StoreInTx(ctx, tx, archivalJobs)
					},
				)
			}, proc.sendRetryStoreStats)
		if err != nil {
			return fmt.Errorf("store into archival table failed with error: %v", err)
		}
		proc.logger.Debug("[Processor] Total jobs written to archiver: ", len(archivalJobs))
		return nil
	})

	g.Go(func() error {
		if len(jobsToTransform) == 0 {
			return nil
		}
		err := misc.RetryWithNotify(
			groupCtx,
			proc.jobsDBCommandTimeout,
			proc.jobdDBMaxRetries,
			func(ctx context.Context) error {
				return proc.transformDB.WithStoreSafeTx(
					ctx,
					func(tx jobsdb.StoreSafeTx) error {
						return proc.transformDB.StoreInTx(ctx, tx, jobsToTransform)
					},
				)
			}, proc.sendRetryStoreStats)
		if err != nil {
			return fmt.Errorf("store into transform table failed with error: %v", err)
		}
		proc.logger.Debug("[Processor] Total jobs written to transform: ", len(jobsToTransform))
		return nil
	})

	g.Go(func() error {
		if len(procErrorJobs) == 0 {
			return nil
		}
		err := misc.RetryWithNotify(
			groupCtx,
			proc.jobsDBCommandTimeout,
			proc.jobdDBMaxRetries,
			func(ctx context.Context) error {
				return proc.writeErrorDB.WithStoreSafeTx(
					ctx,
					func(tx jobsdb.StoreSafeTx) error {
						return proc.writeErrorDB.StoreInTx(ctx, tx, procErrorJobs)
					},
				)
			}, proc.sendRetryStoreStats)
		if err != nil {
			return fmt.Errorf("store into error table failed with error: %v", err)
		}
		proc.logger.Debug("[Processor] Total jobs written to error: ", len(procErrorJobs))
		return nil
	})

	g.Go(func() error {
		if len(statusList) == 0 {
			return nil
		}
		err := misc.RetryWithNotify(
			groupCtx,
			proc.jobsDBCommandTimeout,
			proc.jobdDBMaxRetries,
			func(ctx context.Context) error {
				return proc.gatewayDB.WithUpdateSafeTx(
					ctx,
					func(tx jobsdb.UpdateSafeTx) error {
						if err := proc.gatewayDB.UpdateJobStatusInTx(
							ctx,
							tx,
							statusList,
							[]string{proc.config.GWCustomVal},
							nil,
						); err != nil {
							return err
						}
						// rsources stats
						storeJob.rsourcesStats.JobStatusesUpdated(statusList)
						if err := storeJob.rsourcesStats.Publish(ctx, tx.SqlTx()); err != nil {
							return fmt.Errorf("publishing rsources stats: %w", err)
						}
						if proc.isReportingEnabled() {
							proc.reporting.Report(reportMetrics, tx.SqlTx())
						}
						return nil
					},
				)
			}, proc.sendRetryStoreStats)
		if err != nil {
			return fmt.Errorf("store into status table failed with error: %v", err)
		}
		proc.logger.Debug("[Processor] Total jobs written to status: ", len(statusList))
		return nil
	})

	// TODO: commit dedupKeys

	return g.Wait()
}
