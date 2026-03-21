package httpapi

type enabledStateView struct {
	Enabled int `json:"enabled"`
}

type operationStatusView struct {
	Status string `json:"status"`
}

type deletedCountView struct {
	Deleted int64 `json:"deleted"`
}
