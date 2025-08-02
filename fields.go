package zlog

// Fields represents a collection of Field values that can be cloned.
// This type is used as the data type for the global logger's Log.
type Fields []Field

// Clone creates a deep copy of the Fields slice.
// This implements the pipz.Cloner interface for use with pipz pipelines.
func (f Fields) Clone() Fields {
	if f == nil {
		return nil
	}
	return append(Fields(nil), f...)
}
