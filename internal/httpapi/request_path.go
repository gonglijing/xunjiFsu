package httpapi

import "net/http"

func parseIDOrWriteBadRequest(w http.ResponseWriter, r *http.Request, def APIErrorDef) (int64, bool) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequestDef(w, def)
		return 0, false
	}
	return id, true
}
