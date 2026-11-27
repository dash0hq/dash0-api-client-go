package dash0

// AssetMetadata is implemented by every definition type that can be upserted, so a caller can read the identifiers without switching on the kind.
// Each method is nil-safe and returns "" or false on a nil receiver.
type AssetMetadata interface {
	// GetOrigin returns the dash0.com/origin label, or "" when the type has none or it is unset.
	// It does not say whether origin is an upsert key, UpsertKeys does.
	GetOrigin() string
	// GetID returns the asset ID.
	GetID() string
	// GetName returns the asset name.
	GetName() string
	// UpsertKeys reports which identifiers the API upserts the asset by.
	// When both are true, the origin is preferred.
	UpsertKeys() (byOrigin, byID bool)
}
