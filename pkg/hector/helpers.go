package hector

// Helper functions for common operations

// boolPtr returns a pointer to the given bool value
func boolPtr(b bool) *bool {
	return &b
}

// boolValue returns the bool value or default if nil
func boolValue(b *bool, defaultValue bool) bool {
	if b == nil {
		return defaultValue
	}
	return *b
}
