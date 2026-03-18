package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

const (
	collectWriteQueueCap      = 256
	collectWriteMaxBatchItems = 32
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
		log.Printf("collect data async batch write failed: %v", err)
		for _, item := range batch {
			if err := InsertCollectDataWithOptions(item.data, item.storeHistory); err != nil {
				log.Printf("collect data fallback write failed: %v", err)
			}
		}
	}
	clear(batch)
}

func writeCollectDataBatch(items []collectWriteRequest) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		return InsertCollectDataWithOptions(items[0].data, items[0].storeHistory)
	}

	tx, err := DataDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin collect data batch transaction: %w", err)
	}
	defer tx.Rollback()

	storedHistory := false
	if err := writeCollectDataBatchItemsWithTx(tx, items, &storedHistory); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit collect data batch transaction: %w", err)
	}

	maybeEnforceDataCacheLimit()
	if storedHistory {
		maybeEnforceDataPointsLimit()
		TriggerSyncIfNeeded()
	}
	return nil
}

func collectWriteQueueRunning() bool {
	collectWriteMu.RLock()
	defer collectWriteMu.RUnlock()
	return collectWriteAlive && collectWriteCh != nil
}

func writeCollectDataBatchWithTx(tx *sql.Tx, items []collectWriteRequest) error {
	return writeCollectDataBatchItemsWithTx(tx, items, nil)
}

func writeCollectDataBatchItemsWithTx(tx *sql.Tx, items []collectWriteRequest, storedHistory *bool) error {
	if tx == nil {
		return fmt.Errorf("nil tx")
	}
	cacheStmtCache := newCollectDataStmtCache(tx, collectDataCacheBatchSQLCache.get)
	defer cacheStmtCache.close()

	var historyStmtCache *collectDataStmtCache
	defer func() {
		if historyStmtCache != nil {
			historyStmtCache.close()
		}
	}()
	for _, item := range items {
		if item.storeHistory && historyStmtCache == nil {
			historyStmtCache = newCollectDataStmtCache(tx, collectDataHistoryBatchSQLCache.get)
		}
		if err := insertCollectDataWithOptionsTx(tx, item.data, item.storeHistory, cacheStmtCache, historyStmtCache); err != nil {
			return err
		}
		if storedHistory != nil && item.storeHistory {
			*storedHistory = true
		}
	}
	return nil
}
