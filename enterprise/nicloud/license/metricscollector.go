package license

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/NeuralInverse/cloud/v2/nicloud/entitlements"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

var (
	activeUsersDesc      = prometheus.NewDesc("nicloud_license_active_users", "The number of active users.", nil, nil)
	limitUsersDesc       = prometheus.NewDesc("nicloud_license_limit_users", "The user seats limit based on the active Coder license.", nil, nil)
	userLimitEnabledDesc = prometheus.NewDesc("nicloud_license_user_limit_enabled", "Returns 1 if the current license enforces the user limit.", nil, nil)

	// Metrics for license warnings and errors.
	licenseWarningsDesc = prometheus.NewDesc("nicloud_license_warnings", "The number of active license warnings.", nil, nil)
	licenseErrorsDesc   = prometheus.NewDesc("nicloud_license_errors", "The number of active license errors.", nil, nil)
)

type MetricsCollector struct {
	Entitlements *entitlements.Set
}

var _ prometheus.Collector = new(MetricsCollector)

func (*MetricsCollector) Describe(descCh chan<- *prometheus.Desc) {
	descCh <- activeUsersDesc
	descCh <- limitUsersDesc
	descCh <- userLimitEnabledDesc
	descCh <- licenseWarningsDesc
	descCh <- licenseErrorsDesc
}

func (mc *MetricsCollector) Collect(metricsCh chan<- prometheus.Metric) {
	// Collect user limit metrics.
	mc.collectUserLimit(metricsCh)

	// Collect license warnings and errors metrics.
	mc.collectWarningsAndErrors(metricsCh)
}

func (mc *MetricsCollector) collectUserLimit(metricsCh chan<- prometheus.Metric) {
	userLimitEntitlement, ok := mc.Entitlements.Feature(nicloudsdk.FeatureUserLimit)
	if !ok {
		return
	}

	var enabled float64
	if userLimitEntitlement.Enabled {
		enabled = 1
	}
	metricsCh <- prometheus.MustNewConstMetric(userLimitEnabledDesc, prometheus.GaugeValue, enabled)

	if userLimitEntitlement.Actual != nil {
		metricsCh <- prometheus.MustNewConstMetric(activeUsersDesc, prometheus.GaugeValue, float64(*userLimitEntitlement.Actual))
	}

	if userLimitEntitlement.Limit != nil {
		metricsCh <- prometheus.MustNewConstMetric(limitUsersDesc, prometheus.GaugeValue, float64(*userLimitEntitlement.Limit))
	}
}

func (mc *MetricsCollector) collectWarningsAndErrors(metricsCh chan<- prometheus.Metric) {
	warnings := mc.Entitlements.Warnings()
	errors := mc.Entitlements.Errors()

	metricsCh <- prometheus.MustNewConstMetric(licenseWarningsDesc, prometheus.GaugeValue, float64(len(warnings)))
	metricsCh <- prometheus.MustNewConstMetric(licenseErrorsDesc, prometheus.GaugeValue, float64(len(errors)))
}
