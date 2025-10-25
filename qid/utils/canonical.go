package utils

import (
	"strings"

	"github.com/gosimple/slug"
)

func ValidateCanonical(c string) bool {
	return slug.IsSlug(c) && len(c) <= 255
}

func ValidateResourceCanonical(rc string) bool {
	rcs := strings.Split(rc, ".")
	return ValidateCanonical(rcs[0]) && ValidateCanonical(rcs[1])
}

func ResourceCanonical(rt, rn string) string { return strings.Join([]string{rt, rn}, ".") }
