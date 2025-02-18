package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-server/warehouse/internal/model"

	"github.com/stretchr/testify/require"

	backendconfig "github.com/rudderlabs/rudder-server/backend-config"
	"github.com/rudderlabs/rudder-server/warehouse/internal/service"
	warehouseutils "github.com/rudderlabs/rudder-server/warehouse/utils"
)

type mockRepo struct {
	m   map[string][]string
	err error
}

type mockDestination struct {
	recovered int
}

func (r *mockRepo) InterruptedDestinations(_ context.Context, destinationType string) ([]string, error) {
	return r.m[destinationType], r.err
}

func (d *mockDestination) CrashRecover(_ context.Context) {
	d.recovered += 1
}

func TestRecovery(t *testing.T) {
	testCases := []struct {
		name          string
		whType        string
		destinationID string

		recovery bool

		repoErr error
		wantErr error
	}{
		{
			name:          "interrupted mssql warehouse",
			whType:        warehouseutils.MSSQL,
			destinationID: "1",
			recovery:      true,
		},
		{
			name:          "non-interrupted mssql warehouse",
			whType:        warehouseutils.MSSQL,
			destinationID: "3",
			recovery:      false,
		},
		{
			name:          "interrupted snowflake - skipped - warehouse",
			whType:        warehouseutils.SNOWFLAKE,
			destinationID: "6",
			recovery:      false,
		},

		{
			name:          "repo error",
			whType:        warehouseutils.MSSQL,
			destinationID: "1",
			repoErr:       fmt.Errorf("repo error"),
			wantErr:       fmt.Errorf("repo interrupted destinations: repo error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{
				m: map[string][]string{
					warehouseutils.MSSQL:     {"1", "2"},
					warehouseutils.SNOWFLAKE: {"6", "8"},
				},
				err: tc.repoErr,
			}

			d := &mockDestination{}

			recovery := service.NewRecovery(tc.whType, repo)

			for i := 0; i < 2; i++ {
				err := recovery.Recover(context.Background(), d, model.Warehouse{
					Destination: backendconfig.DestinationT{
						ID: tc.destinationID,
					},
				})

				if tc.wantErr != nil {
					require.EqualError(t, err, tc.wantErr.Error())
				} else {
					require.NoError(t, err)
				}
			}

			if tc.recovery {
				require.Equal(t, 1, d.recovered)
			} else {
				require.Equal(t, 0, d.recovered)
			}
		})
	}
}
