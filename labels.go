package dash0

// CRD label keys used by Dash0 assets in Kubernetes-style metadata.
const (
	LabelID      = "dash0.com/id"
	LabelDataset = "dash0.com/dataset"
	LabelOrigin  = "dash0.com/origin"
	// LabelSource records which system last wrote the asset.
	// It has the same value as [AnnotationSource], which some assets carry as an annotation instead.
	LabelSource = "dash0.com/source"
	// LabelVersion is the server-set version used for optimistic concurrency on update.
	LabelVersion = "dash0.com/version"
)
