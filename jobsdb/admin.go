package jobsdb

import (
	"context"
	"fmt"
	"time"

	"github.com/rudderlabs/rudder-server/utils/misc"
)

/*
Ping returns health check for pg database
*/
func (jd *Handle) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := jd.dbHandle.ExecContext(ctx, `SELECT 'Rudder DB Health Check'::text as message`)
	if err != nil {
		return err
	}
	return nil
}

/*
DeleteExecuting deletes events whose latest job state is executing.
This is only done during recovery, which happens during the server start.
*/
func (jd *Handle) DeleteExecuting() {
	tags := statTags{CustomValFilters: []string{jd.tablePrefix}}
	command := func() any {
		jd.deleteJobStatus()
		return nil
	}
	_ = executeDbRequest(jd, newWriteDbRequest("delete_job_status", &tags, command))
}

// deleteJobStatus deletes the latest status of a batch of jobs
func (jd *Handle) deleteJobStatus() {
	err := jd.WithUpdateSafeTx(context.TODO(), func(tx UpdateSafeTx) error {
		defer jd.getTimerStat(
			"jobsdb_delete_job_status_time",
			&statTags{
				CustomValFilters: []string{jd.tablePrefix},
			}).RecordDuration()()

		dsList := jd.getDSList()

		for _, ds := range dsList {
			ds := ds
			if err := jd.deleteJobStatusDSInTx(tx.SqlTx(), ds); err != nil {
				return err
			}
			tx.Tx().AddSuccessListener(func() {
				jd.noResultsCache.InvalidateDataset(ds.Index)
			})
		}

		return nil
	})
	jd.assertError(err)
}

func (jd *Handle) deleteJobStatusDSInTx(txHandler transactionHandler, ds dataSetT) error {
	defer jd.getTimerStat(
		"jobsdb_delete_job_status_ds_time",
		&statTags{
			CustomValFilters: []string{jd.tablePrefix},
		}).RecordDuration()()

	_, err := txHandler.Exec(
		fmt.Sprintf(
			`DELETE FROM %[1]q
				WHERE id = ANY(
					SELECT id from "v_last_%[1]s" where job_state='executing'
				)`,
			ds.JobStatusTable,
		),
	)
	return err
}

/*
FailExecuting fails events whose latest job state is executing.

This is only done during recovery, which happens during the server start.
*/
func (jd *Handle) FailExecuting() {
	tags := statTags{
		CustomValFilters: []string{jd.tablePrefix},
	}
	command := func() any {
		jd.failExecuting()
		return nil
	}
	_ = executeDbRequest(jd, newWriteDbRequest("fail_executing", &tags, command))
}

// failExecuting sets the state of the executing jobs to failed
func (jd *Handle) failExecuting() {
	err := jd.WithUpdateSafeTx(context.TODO(), func(tx UpdateSafeTx) error {
		defer jd.getTimerStat(
			"jobsdb_fail_executing_time",
			&statTags{CustomValFilters: []string{jd.tablePrefix}},
		).RecordDuration()()

		dsList := jd.getDSList()

		for _, ds := range dsList {
			ds := ds
			err := jd.failExecutingDSInTx(tx.SqlTx(), ds)
			if err != nil {
				return err
			}
			tx.Tx().AddSuccessListener(func() {
				jd.noResultsCache.InvalidateDataset(ds.Index)
			})
		}
		return nil
	})
	jd.assertError(err)
}

func (jd *Handle) failExecutingDSInTx(txHandler transactionHandler, ds dataSetT) error {
	defer jd.getTimerStat(
		"jobsdb_fail_executing_ds_time",
		&statTags{CustomValFilters: []string{jd.tablePrefix}},
	).RecordDuration()()

	_, err := txHandler.Exec(
		fmt.Sprintf(
			`UPDATE %[1]q SET job_state='failed'
				WHERE id = ANY(
					SELECT id from "v_last_%[1]s" where job_state='executing'
				)`,
			ds.JobStatusTable,
		),
	)
	return err
}

func (jd *Handle) startCleanupLoop(ctx context.Context) {
	jd.backgroundGroup.Go(misc.WithBugsnag(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-jd.TriggerJobCleanUp():
				func() {
					for {
						if err := jd.doCleanup(ctx, jd.config.GetInt("jobsdb.cleanupBatchSize", 100)); err != nil && ctx.Err() == nil {
							jd.logger.Errorf("error while cleaning up old jobs: %w", err)
							if err := misc.SleepCtx(ctx, jd.config.GetDuration("jobsdb.cleanupRetryInterval", 10, time.Second)); err != nil {
								return
							}
							continue
						}
						return
					}
				}()
			}
		}
	}))
}

func (jd *Handle) doCleanup(ctx context.Context, batchSize int) error {
	// 1. cleanup journal
	{
		deleteStmt := "DELETE FROM %s_journal WHERE start_time < NOW() - INTERVAL '%d DAY'"
		var journalEntryCount int64
		res, err := jd.dbHandle.ExecContext(
			ctx,
			fmt.Sprintf(
				deleteStmt,
				jd.tablePrefix,
				jd.config.GetInt("JobsDB.archivalTimeInDays", 10),
			),
		)
		if err != nil {
			return fmt.Errorf("cleaning up journal: %w", err)
		}
		journalEntryCount, err = res.RowsAffected()
		if err != nil {
			return fmt.Errorf("finding journal rows affected during cleanup: %w", err)
		}
		jd.logger.Infof("cleaned up %d journal entries", journalEntryCount)
	}

	return nil
}
