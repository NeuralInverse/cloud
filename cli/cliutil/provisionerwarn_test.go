package cliutil_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/cliutil"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestWarnMatchedProvisioners(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		mp     *nicloudsdk.MatchedProvisioners
		job    nicloudsdk.ProvisionerJob
		expect string
	}{
		{
			name: "no_match",
			mp: &nicloudsdk.MatchedProvisioners{
				Count:     0,
				Available: 0,
			},
			job: nicloudsdk.ProvisionerJob{
				Status: nicloudsdk.ProvisionerJobPending,
			},
			expect: `there are no provisioners that accept the required tags`,
		},
		{
			name: "no_available",
			mp: &nicloudsdk.MatchedProvisioners{
				Count:     1,
				Available: 0,
			},
			job: nicloudsdk.ProvisionerJob{
				Status: nicloudsdk.ProvisionerJobPending,
			},
			expect: `Provisioners that accept the required tags have not responded for longer than expected`,
		},
		{
			name: "match",
			mp: &nicloudsdk.MatchedProvisioners{
				Count:     1,
				Available: 1,
			},
			job: nicloudsdk.ProvisionerJob{
				Status: nicloudsdk.ProvisionerJobPending,
			},
		},
		{
			name: "not_pending",
			mp:   &nicloudsdk.MatchedProvisioners{},
			job: nicloudsdk.ProvisionerJob{
				Status: nicloudsdk.ProvisionerJobRunning,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var w strings.Builder
			cliutil.WarnMatchedProvisioners(&w, tt.mp, tt.job)
			if tt.expect != "" {
				require.Contains(t, w.String(), tt.expect)
			} else {
				require.Empty(t, w.String())
			}
		})
	}
}
