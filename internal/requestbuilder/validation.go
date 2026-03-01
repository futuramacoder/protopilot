package requestbuilder

// ValidateField performs live validation on a single field.
// Returns an error message or empty string.
func ValidateField(field *FormField) string {
	if field.Widget == nil {
		return ""
	}
	return field.Widget.Validate()
}

// ValidateAll validates all fields in the form.
// Returns a list of (path, error) pairs for invalid fields.
func ValidateAll(fields []FormField) []ValidationError {
	var errs []ValidationError
	validateRecursive(fields, &errs)
	return errs
}

// ValidationError represents a validation failure for a specific field.
type ValidationError struct {
	Path    string
	Message string
}

func validateRecursive(fields []FormField, errs *[]ValidationError) {
	for i := range fields {
		f := &fields[i]

		// Skip inactive oneof variants.
		if f.OneofGroup != "" && !f.OneofActive {
			continue
		}

		if f.Widget != nil {
			if msg := f.Widget.Validate(); msg != "" {
				*errs = append(*errs, ValidationError{
					Path:    f.Info.Path,
					Message: msg,
				})
			}
		}

		// Validate children recursively.
		if len(f.Children) > 0 {
			validateRecursive(f.Children, errs)
		}
	}
}
