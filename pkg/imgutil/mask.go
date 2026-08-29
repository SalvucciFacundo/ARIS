package imgutil

import (
	"fmt"
)

// MaskConfig encapsulates mask parameters and validation.
type MaskConfig struct {
	MaskPath string
	Invert   bool
	Blur     int
}

// ValidateMask ensures that a mask file exists, is a supported format, and matches dimensions of base image.
func ValidateMask(basePath, maskPath string, maxSize int64) error {
	if maskPath == "" {
		return nil
	}
	if basePath == "" {
		return fmt.Errorf("mask provided without base image path")
	}

	baseData, _, err := LoadAndValidateImage(basePath, maxSize)
	if err != nil {
		return fmt.Errorf("failed to load base image: %w", err)
	}

	maskData, _, err := LoadAndValidateImage(maskPath, maxSize)
	if err != nil {
		return fmt.Errorf("failed to load mask image: %w", err)
	}

	return ValidateMatchingDimensions(baseData, maskData)
}
