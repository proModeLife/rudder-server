package isolation

import (
	"context"
	"errors"
	"strings"

	"github.com/rudderlabs/rudder-server/jobsdb"
)

type Mode string

const (
	ModeNone       Mode = "none"
	ModeWorkspace  Mode = "workspace"
	ModeSource     Mode = "source"
	ModeConnection Mode = "connection"
)

// GetStrategy returns the strategy for the given isolation mode. An error is returned if the mode is invalid
func GetStrategy(mode Mode) (Strategy, error) {
	switch mode {
	case ModeNone:
		return noneStrategy{}, nil
	case ModeWorkspace:
		return workspaceStrategy{}, nil
	case ModeSource:
		return sourceStrategy{}, nil
	case ModeConnection:
		return connection{}, nil
	default:
		return noneStrategy{}, errors.New("unsupported isolation mode")
	}
}

// Strategy defines the operations that every different isolation strategy in processor must implement
type Strategy interface {
	// ActivePartitions returns the list of partitions that are active for the given strategy
	ActivePartitions(ctx context.Context, db jobsdb.JobsDB) ([]string, error)
	// AugmentQueryParams augments the given GetQueryParamsT with the strategy specific parameters
	AugmentQueryParams(partition string, params *jobsdb.GetQueryParams)
}

// noneStrategy implements isolation at no level
type noneStrategy struct{}

func (noneStrategy) ActivePartitions(_ context.Context, _ jobsdb.JobsDB) ([]string, error) {
	return []string{""}, nil
}

func (noneStrategy) AugmentQueryParams(_ string, _ *jobsdb.GetQueryParams) {
	// no-op
}

// workspaceStrategy implements isolation at workspace level
type workspaceStrategy struct{}

// ActivePartitions returns the list of active workspaceIDs in jobsdb
func (workspaceStrategy) ActivePartitions(ctx context.Context, db jobsdb.JobsDB) ([]string, error) {
	return db.GetActiveWorkspaces(ctx, "")
}

func (workspaceStrategy) AugmentQueryParams(partition string, params *jobsdb.GetQueryParams) {
	params.WorkspaceID = partition
}

// sourceStrategy implements isolation at source level
type sourceStrategy struct{}

// ActivePartitions returns the list of active sourceIDs in jobsdb
func (sourceStrategy) ActivePartitions(ctx context.Context, db jobsdb.JobsDB) ([]string, error) {
	return db.GetDistinctParameterValues(ctx, "source_id")
}

// AugmentQueryParams augments the given GetQueryParamsT by adding the partition as sourceID parameter filter
func (sourceStrategy) AugmentQueryParams(partition string, params *jobsdb.GetQueryParams) {
	params.ParameterFilters = append(params.ParameterFilters, jobsdb.ParameterFilterT{Name: "source_id", Value: partition})
}

type connection struct{}

func (connection) ActivePartitions(ctx context.Context, db jobsdb.JobsDB) ([]string, error) {
	return db.GetDistinctConnections(ctx)
}

func (connection) AugmentQueryParams(partition string, params *jobsdb.GetQueryParams) {
	fields := strings.Split(partition, "::")
	params.ParameterFilters = append(
		params.ParameterFilters,
		jobsdb.ParameterFilterT{Name: "source_id", Value: fields[0]},
		jobsdb.ParameterFilterT{Name: "destination_id", Value: fields[1]},
	)
}
