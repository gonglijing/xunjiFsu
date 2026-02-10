package adapters

import "github.com/gonglijing/xunjiFsu/internal/models"

func appendCommandQueueWithCap(queue []*models.NorthboundCommand, incoming []*models.NorthboundCommand, capLimit int) []*models.NorthboundCommand {
	if capLimit <= 0 {
		capLimit = defaultRealtimeQueue
	}

	if len(queue) > capLimit {
		overflow := len(queue) - capLimit
		clear(queue[:overflow])
		queue = queue[overflow:]
	}

	if len(incoming) == 0 {
		return queue[:len(queue):len(queue)]
	}

	validIncoming := 0
	for _, item := range incoming {
		if item != nil {
			validIncoming++
		}
	}
	if validIncoming == 0 {
		return queue[:len(queue):len(queue)]
	}

	drop := len(queue) + validIncoming - capLimit
	skipIncoming := 0
	if drop > 0 {
		dropExisting := drop
		if dropExisting > len(queue) {
			dropExisting = len(queue)
		}
		if dropExisting > 0 {
			clear(queue[:dropExisting])
			queue = queue[dropExisting:]
		}
		skipIncoming = drop - dropExisting
	}

	for _, item := range incoming {
		if item == nil {
			continue
		}
		if skipIncoming > 0 {
			skipIncoming--
			continue
		}
		queue = append(queue, item)
	}

	if len(queue) > capLimit {
		overflow := len(queue) - capLimit
		clear(queue[:overflow])
		queue = queue[overflow:]
	}

	return queue[:len(queue):len(queue)]
}
