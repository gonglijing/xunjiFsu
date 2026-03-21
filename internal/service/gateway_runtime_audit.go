package service

import (
	"encoding/json"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

func (s *GatewayRuntimeService) ListRuntimeConfigAudits(limit int) ([]*database.RuntimeConfigAudit, error) {
	return database.ListRuntimeConfigAudits(limit)
}

func (s *GatewayRuntimeService) RecordRuntimeConfigAudit(actor RuntimeConfigActor, changes map[string]RuntimeConfigChange) error {
	if len(changes) == 0 {
		return nil
	}

	changeJSON, err := json.Marshal(changes)
	if err != nil {
		return err
	}

	username := strings.TrimSpace(actor.Username)
	if username == "" {
		username = "unknown"
	}

	_, err = database.CreateRuntimeConfigAudit(&database.RuntimeConfigAudit{
		OperatorUserID:   actor.UserID,
		OperatorUsername: username,
		SourceIP:         strings.TrimSpace(actor.SourceIP),
		Changes:          string(changeJSON),
	})
	return err
}
