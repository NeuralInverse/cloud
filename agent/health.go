package agent

import (
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/healthcheck/health"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/healthsdk"
)

func (a *agent) HandleNetcheck(rw http.ResponseWriter, r *http.Request) {
	ni := a.TailnetConn().GetNetInfo()

	ifReport, err := healthsdk.RunInterfacesReport()
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to run interfaces report",
			Detail:  err.Error(),
		})
		return
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, healthsdk.AgentNetcheckReport{
		BaseReport: healthsdk.BaseReport{
			Severity: health.SeverityOK,
		},
		NetInfo:    ni,
		Interfaces: ifReport,
	})
}
