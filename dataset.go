package dash0

// DatasetPtr converts a dataset string to a pointer, returning nil for empty
// strings and for "default" (the API uses "default" implicitly when no dataset
// parameter is sent).
func DatasetPtr(dataset string) *string {
	if dataset == "" || dataset == "default" {
		return nil
	}
	return &dataset
}
