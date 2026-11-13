package dash0

// CRD annotation keys used by Dash0 assets in Kubernetes-style metadata.
const (
	AnnotationFolderPath = "dash0.com/folder-path"
	AnnotationSharing    = "dash0.com/sharing"
	AnnotationSource     = "dash0.com/source"
)

// Annotation keys on a rule in a PrometheusRule CRD that map to typed fields of a [PrometheusAlertRule].
const (
	// AnnotationPrometheusRuleEnabled is "false" for a disabled rule.
	// It differs from the dash0.com/enabled annotation of other assets.
	AnnotationPrometheusRuleEnabled = "dash0-enabled"

	AnnotationThresholdDegraded = "dash0-threshold-degraded"
	AnnotationThresholdCritical = "dash0-threshold-critical"
	// AnnotationThresholdDegradedLegacy is the unprefixed spelling of [AnnotationThresholdDegraded], still accepted on import.
	AnnotationThresholdDegradedLegacy = "threshold-degraded"
	// AnnotationThresholdCriticalLegacy is the unprefixed spelling of [AnnotationThresholdCritical], still accepted on import.
	AnnotationThresholdCriticalLegacy = "threshold-critical"

	AnnotationBaselineDirection        = "dash0.com/baseline-direction"
	AnnotationBaselineSpreadFloor      = "dash0.com/baseline-spread-floor"
	AnnotationVolumeFloor              = "dash0.com/volume-floor"
	AnnotationChangeGateComparison     = "dash0.com/change-gate-comparison"
	AnnotationChangeGateValue          = "dash0.com/change-gate-value"
	AnnotationChangeGateBaselineWindow = "dash0.com/change-gate-baseline-window"
)
