package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

const (
	collectWriteQueueCap      = 64
	collectWriteMaxBatchItems = 16
)

type collectWriteRequest struct {
	data         *models.CollectData
	storeHistory bool
}

var (
	collectWriteMu    sync.RWMutex
	collectWriteCh    chan collectWriteRequest
	collectWriteWG    sync.WaitGroup
	collectWriteAlive bool
)

func StartCollectDataWriter() {
	collectWriteMu.Lock()
	defer collectWriteMu.Unlock()
	if collectWriteAlive {
		return
	}
	collectWriteCh = make(chan collectWriteRequest, collectWriteQueueCap)
	collectWriteAlive = true
	collectWriteWG.Add(1)
	go runCollectDataWriter(collectWriteCh)
}

func StopCollectDataWriter() {
	collectWriteMu.Lock()
	if !collectWriteAlive {
		collectWriteMu.Unlock()
		return
	}
	ch := collectWriteCh
	collectWriteCh = nil
	collectWriteAlive = false
	close(ch)
	collectWriteMu.Unlock()

	collectWriteWG.Wait()
}

func EnqueueCollectDataWrite(data *models.CollectData, storeHistory bool) error {
	if data == nil {
		return nil
	}

	collectWriteMu.RLock()
	ch := collectWriteCh
	running := collectWriteAlive
	if !running || ch == nil {
		collectWriteMu.RUnlock()
		return InsertCollectDataWithOptions(data, storeHistory)
	}

	select {
	case ch <- collectWriteRequest{data: data, storeHistory: storeHistory}:
		collectWriteMu.RUnlock()
		return nil
	default:
		collectWriteMu.RUnlock()
		return InsertCollectDataWithOptions(data, storeHistory)
	}
}

func runCollectDataWriter(ch <-chan collectWriteRequest) {
	defer collectWriteWG.Done()

	batch := make([]collectWriteRequest, 0, collectWriteMaxBatchItems)

	for item := range ch {
		batch = append(batch, item)
		for len(batch) < collectWriteMaxBatchItems {
			select {
			case next, ok := <-ch:
				if !ok {
					flushCollectDataBatch(batch)
					return
				}
				batch = append(batch, next)
			default:
				flushCollectDataBatch(batch)
				clear(batch)
				batch = batch[:0]
				goto nextLoop
			}
		}
		flushCollectDataBatch(batch)
		clear(batch)
		batch = batch[:0]
	nextLoop:
	}
	flushCollectDataBatch(batch)
}

func flushCollectDataBatch(batch []collectWriteRequest) {
	if len(batch) == 0 {
		return
	}
	if err := writeCollectDataBatch(batch); err != nil {
		slog.Error("collect data async batch write failed", "error", err)
		for _, item := range batch {
			if err := InsertCollectDataWithOptions(item.data, item.storeHistory); err != nil {
				slog.Error("collect data fallback write failed", "error", err)
			}
		}
	}
}

func writeCollectDataBatch(items []collectWriteRequest) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		return InsertCollectDataWithOptions(items[0].data, items[0].storeHistory)
	}
	if err := writeCollectDataCacheOnlyBatchDirect(items); err == nil {
		maybeEnforceDataCacheLimit()
		return nil
	}

	tx, err := DataDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin collect data batch transaction: %w", err)
	}
	defer tx.Rollback()

	var historyRows int
	if err := writeCollectDataBatchItemsWithTx(tx, items, &historyRows); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit collect data batch transaction: %w", err)
	}

	maybeEnforceDataCacheLimit()
	if historyRows > 0 {
		noteHistoryRowsWritten(historyRows)
		maybeEnforceDataPointsLimit()
		TriggerSyncIfNeeded()
	}
	return nil
}

func writeCollectDataCacheOnlyBatchDirect(items []collectWriteRequest) error {
	args := getCollectDataArgs(collectDataCacheBatchSize * 3)
	defer putCollectDataArgs(args)
	batchRows := 0

	flush := func() error {
		if batchRows == 0 {
			return nil
		}
		stmt, err := dataCacheExecStmtCache.get(DataDB, collectDataCacheStringBatchSQLCache.get(batchRows))
		if err != nil {
			return fmt.Errorf("failed to prepare data cache batch statement: %w", err)
		}
		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("failed to upsert data cache batch: %w", err)
		}
		args = args[:0]
		batchRows = 0
		return nil
	}

	for _, item := range items {
		if item.storeHistory {
			return fmt.Errorf("history write present")
		}
		shape := buildCollectDataCacheShape(item.data)
		rows := shape.rows
		if rows == 0 {
			shape.release()
			continue
		}
		if rows > collectDataCacheBatchSize {
			shape.release()
			return fmt.Errorf("cache item too large")
		}
		if batchRows > 0 && batchRows+rows > collectDataCacheBatchSize {
			if err := flush(); err != nil {
				shape.release()
				return err
			}
		}
		args = appendCollectDataCacheArgsForDataWithShape(args, item.data, shape)
		shape.release()
		batchRows += rows
	}
	return flush()
}

func writeCollectDataBatchItemsWithTx(tx *sql.Tx, items []collectWriteRequest, historyRows *int) error {
	if tx == nil {
		return fmt.Errorf("nil tx")
	}
	cacheStmtCache := newCollectDataStmtCache(tx, collectDataCacheStringBatchSQLCache.get)
	defer cacheStmtCache.close()

	var historyStmtCache *collectDataStmtCache
	defer func() {
		if historyStmtCache != nil {
			historyStmtCache.close()
		}
	}()
	for _, item := range items {
		if item.storeHistory && historyStmtCache == nil {
			historyStmtCache = newCollectDataStmtCache(tx, collectDataHistoryStringBatchSQLCache.get)
		}
		written, err := insertCollectDataWithOptionsTx(tx, item.data, item.storeHistory, cacheStmtCache, historyStmtCache)
		if err != nil {
			return err
		}
		if historyRows != nil {
			*historyRows += written
		}
	}
	return nil
}
