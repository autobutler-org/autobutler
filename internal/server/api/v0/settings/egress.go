package v0_settings

import (
	"github.com/autobutler-org/quark/pkg/util/egressutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// egressRuleJSON is the API representation of a single egress rule.
type egressRuleJSON struct {
	Host    string `json:"host"`
	Port    string `json:"port"`
	Proto   string `json:"proto"`
	Comment string `json:"comment"`
}

// egressStatusJSON is the API representation of an applied-rule status entry.
type egressStatusJSON struct {
	Rule    string `json:"rule"`
	Applied bool   `json:"applied"`
}

// egressResponse is the top-level response for the update-egress endpoints.
type egressResponse struct {
	Backend string             `json:"backend"`
	Rules   []egressRuleJSON   `json:"rules"`
	Status  []egressStatusJSON `json:"status,omitempty"`
}

// getUpdateEgress godoc
// @Summary Get update-egress allowlist
// @Description Returns the canonical set of outbound firewall rules needed for software updates.
// @Description When ufw is available, also returns whether each rule is currently applied.
// @Tags settings
// @Produce json
// @Success 200 {object} egressResponse
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /settings/update-egress [get]
var getUpdateEgressRoute = serverutil.ApiRoute(
	"GET", "/settings/update-egress",
	func(c *gin.Context) *serverutil.Response {
		backend := egressutil.DetectBackend()
		rules := egressutil.UpdateEgressRules()

		jsonRules := make([]egressRuleJSON, len(rules))
		for i, r := range rules {
			jsonRules[i] = egressRuleJSON{
				Host:    r.Host,
				Port:    r.Port,
				Proto:   r.Proto,
				Comment: r.Comment,
			}
		}

		resp := egressResponse{
			Backend: string(backend),
			Rules:   jsonRules,
		}

		// If ufw is available, include the live applied-status query.
		if backend == egressutil.BackendUFW {
			entries, err := egressutil.QueryUpdateEgressStatus()
			if err == nil {
				jsonStatus := make([]egressStatusJSON, len(entries))
				for i, e := range entries {
					jsonStatus[i] = egressStatusJSON{Rule: e.Rule, Applied: e.Applied}
				}
				resp.Status = jsonStatus
			}
		}

		return serverutil.Ok().WithData(resp)
	},
)

// applyUpdateEgress godoc
// @Summary Apply update-egress allowlist rules via ufw
// @Description Applies the outbound firewall allowlist rules for software updates.
// @Description Requires root privileges (or sudo configured for the quark process).
// @Description Rules are idempotent — safe to call multiple times.
// @Tags settings
// @Produce json
// @Success 200 {object} egressResponse
// @Failure 500 {object} serverutil.Response "ufw not available or rule application failed"
// @Security BearerAuth
// @Router /settings/update-egress/apply [post]
var applyUpdateEgressRoute = serverutil.ApiRoute(
	"POST", "/settings/update-egress/apply",
	func(c *gin.Context) *serverutil.Response {
		if err := egressutil.ApplyUpdateEgressRules(); err != nil {
			return serverutil.InternalServerError(err)
		}

		// Return the post-apply status.
		backend := egressutil.DetectBackend()
		rules := egressutil.UpdateEgressRules()
		jsonRules := make([]egressRuleJSON, len(rules))
		for i, r := range rules {
			jsonRules[i] = egressRuleJSON{
				Host:    r.Host,
				Port:    r.Port,
				Proto:   r.Proto,
				Comment: r.Comment,
			}
		}

		resp := egressResponse{
			Backend: string(backend),
			Rules:   jsonRules,
		}

		if backend == egressutil.BackendUFW {
			entries, err := egressutil.QueryUpdateEgressStatus()
			if err == nil {
				jsonStatus := make([]egressStatusJSON, len(entries))
				for i, e := range entries {
					jsonStatus[i] = egressStatusJSON{Rule: e.Rule, Applied: e.Applied}
				}
				resp.Status = jsonStatus
			}
		}

		return serverutil.Ok().WithData(resp)
	},
)
