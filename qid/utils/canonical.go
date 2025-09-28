package utils

import (
	"github.com/gosimple/slug"
)

func ValidateCanonical(c string) bool {
	return slug.IsSlug(c) && len(c) <= 255
}
