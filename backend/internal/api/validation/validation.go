package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

}

// ValidateStruct validates struct fields and uses JSON tag names for error keys.
func ValidateStruct(s interface{}) map[string][]string {
	errors := make(map[string][]string)

	// Validate the struct using the go-playground validator.
	err := validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			// Get the full namespace (e.g., CreateItemsRequest.Items[0].Name)
			namespace := err.StructNamespace()

			// Split the namespace into parts and remove the root.
			parts := strings.Split(namespace, ".")
			if len(parts) > 0 {
				parts = parts[1:]
			}

			// Build a JSON path based on JSON tags.
			jsonPath := buildJSONPath(s, parts)
			if jsonPath == "" {
				jsonPath = err.Field() // Fallback to field name
			}

			if _, exists := errors[jsonPath]; !exists {
				errors[jsonPath] = make([]string, 0)
			}
			errors[jsonPath] = append(errors[jsonPath], getErrorMsg(err))
		}
	}

	return errors
}

// ValidateCreateFolderStructureRequest validates the CreateFolderStructureRequest
func ValidateCreateFolderStructureRequest(request interface{}) error {
	validationErrors := ValidateStruct(request)
	if len(validationErrors) > 0 {
		return fmt.Errorf("%s", formatValidationErrors(validationErrors))
	}
	return nil
}

// formatValidationErrors formats validation errors into a human-readable string
func formatValidationErrors(errors map[string][]string) string {
	var errorMessages []string
	for field, messages := range errors {
		for _, message := range messages {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", field, message))
		}
	}
	return strings.Join(errorMessages, "; ")
}

// buildJSONPath constructs a JSON path from struct namespace parts using JSON tags.
func buildJSONPath(s interface{}, parts []string) string {
	var jsonParts []string
	t := reflect.TypeOf(s)

	// Dereference pointer if needed.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for _, part := range parts {
		var fieldName, index string
		if strings.Contains(part, "[") {
			fieldName = part[:strings.Index(part, "[")]
			index = part[strings.Index(part, "[")+1 : strings.Index(part, "]")]
		} else {
			fieldName = part
		}

		// Retrieve JSON tag from the struct field.
		jsonTag, found := getJSONField(t, fieldName)
		if !found {
			jsonTag = strings.ToLower(fieldName)
		}
		jsonParts = append(jsonParts, jsonTag)

		// Append index if present.
		if index != "" {
			jsonParts[len(jsonParts)-1] += "." + index
		}

		// Set t to the type of the current field for further traversal.
		structField, exists := t.FieldByName(fieldName)
		if !exists {
			break
		}
		fieldType := structField.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			t = fieldType
		} else {
			break
		}
	}

	return strings.Join(jsonParts, ".")
}

// getJSONField retrieves the JSON tag for a given struct field.
func getJSONField(t reflect.Type, fieldName string) (string, bool) {
	field, found := t.FieldByName(fieldName)
	if !found {
		return "", false
	}
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return "", false
	}
	// Return only the first part of the tag (ignore options like omitempty).
	jsonTag = strings.Split(jsonTag, ",")[0]
	return jsonTag, true
}

func getErrorMsg(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("'%s' field is required.", err.StructField())
	case "email":
		return fmt.Sprintf("'%v' is not a valid email address.", err.Value())
	case "min":
		if err.Kind().String() == "slice" {
			return fmt.Sprintf("At least %s item(s) required.", err.Param())
		}
		return fmt.Sprintf("Must be at least %s characters long.", err.Param())
	case "max":
		return fmt.Sprintf("Must not exceed %s characters.", err.Param())
	case "url":
		return fmt.Sprintf("'%v' is not a valid URL.", err.Value())
	case "oneof":
		return fmt.Sprintf("Invalid value '%v'; must be one of %s.", err.Value(), err.Param())
	case "datetime":
		return fmt.Sprintf("'%v' is not a valid date and time.", err.Value())
	case "startswith":
		return fmt.Sprintf("'%v' must start with %s.", err.Value(), err.Param())
	default:
		return fmt.Sprintf("Failed validation on '%s' with tag '%s'.", err.StructField(), err.Tag())
	}
}
