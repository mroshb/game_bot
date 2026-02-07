package utils

import "strings"

// NormalizePersianNumbers converts Persian and Arabic numerals to English numerals
func NormalizePersianNumbers(input string) string {
	replacer := strings.NewReplacer(
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	)
	return replacer.Replace(input)
}

// NormalizePersianText handles Arabic/Farsi character variants
func NormalizePersianText(input string) string {
	replacer := strings.NewReplacer(
		"ي", "ی", // Arabic Yeh to Farsi Yeh
		"ك", "ک", // Arabic Kaf to Farsi Kaf
		"ة", "ه", // Teh Marbuta to Heh
	)
	return strings.TrimSpace(replacer.Replace(input))
}
