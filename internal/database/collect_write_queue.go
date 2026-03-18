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
	flush := func() {
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
		batch = batch[:0]
	}

	for item := range ch {
		batch = append(batch, item)
		for len(batch) < collectWriteMaxBatchItems {
			select {
			case next, ok := <-ch:
				if !ok {
					flush()
					return
				}
				batch = append(batch, next)
			default:
				flush()
				goto nextLoop
			}
		}
		flush()
	nextLoop:
	}
	flush()
}

func writeCollectDataBatch(items []collectWriteRequest) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := DataDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin collect data batch transaction: %w", err)
	}
	defer tx.Rollback()

	cacheStmtCache := newCollectDataStmtCache(tx, collectDataCacheBatchSQLCache.get)
	defer cacheStmtCache.close()
	historyStmtCache := newCollectDataStmtCache(tx, collectDataHistoryBatchSQLCache.get)
	defer historyStmtCache.close()

	storedHistory := false
	for _, item := range items {
		if err := insertCollectDataWithOptionsTx(tx, item.data, item.storeHistory, cacheStmtCache, historyStmtCache); err != nil {
			return err
		}
		if item.storeHistory {
			storedHistory = true
		}
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
	if tx == nil {
		return fmt.Errorf("nil tx")
	}
	cacheStmtCache := newCollectDataStmtCache(tx, collectDataCacheBatchSQLCache.get)
	defer cacheStmtCache.close()
	historyStmtCache := newCollectDataStmtCache(tx, collectDataHistoryBatchSQLCache.get)
	defer historyStmtCache.close()
	for _, item := range items {
		if err := insertCollectDataWithOptionsTx(tx, item.data, item.storeHistory, cacheStmtCache, historyStmtCache); err != nil {
			return err
		}
	}
	return nil
}
