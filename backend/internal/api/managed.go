package api

import (
	"net/http"

	"github.com/netglance/netglance/internal/config"
)

// managedHandler exposes whether netglance is running under an external
// orchestrator (the OPNsense plugin sets NETGLANCE_MANAGED=1) and which
// settings keys it owns. The frontend uses the response to render those
// fields read-only with a "Managed by OPNsense" badge.
func managedHandler(managed bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fields := []string{}
		if managed {
			fields = config.ManagedFields
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"managed": managed,
			"fields":  fields,
		})
	}
}
