package security

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var (
	htmlPolicy = bluemonday.StrictPolicy()
	phoneRegex = regexp.MustCompile(`^[0-9]{10,15}$`)
)

// SanitizeString removes potentially dangerous characters
func SanitizeString(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Limit length
	if len(input) > 1000 {
		input = input[:1000]
	}

	return input
}

// SanitizeHTML removes all HTML tags
func SanitizeHTML(input string) string {
	return htmlPolicy.Sanitize(input)
}

// ValidatePhoneNumber checks if phone number is valid
func ValidatePhoneNumber(phone string) bool {
	// Remove common separators
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "+", "")

	return phoneRegex.MatchString(phone)
}

// ValidateAge checks if age is within valid range
func ValidateAge(age int) bool {
	return age >= 13 && age <= 100
}

// ValidateFileType checks if file extension is allowed
func ValidateFileType(filename string, allowedTypes []string) bool {
	filename = strings.ToLower(filename)
	for _, ext := range allowedTypes {
		if strings.HasSuffix(filename, strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

// ValidateFileSize checks if file size is within limit
func ValidateFileSize(size int64, maxSize int64) bool {
	return size > 0 && size <= maxSize
}
