package domain_test

import (
	"testing"

	"aris/internal/core/domain"
)

func TestReferenceMode_Constants(t *testing.T) {
	if domain.ModeText2Img != "text2img" {
		t.Errorf("expected text2img, got %s", domain.ModeText2Img)
	}
	if domain.ModeImg2Img != "img2img" {
		t.Errorf("expected img2img, got %s", domain.ModeImg2Img)
	}
	if domain.ModeInpaint != "inpaint" {
		t.Errorf("expected inpaint, got %s", domain.ModeInpaint)
	}
	if domain.ModeStyleTransfer != "style_transfer" {
		t.Errorf("expected style_transfer, got %s", domain.ModeStyleTransfer)
	}
	if domain.ModeUpscale != "upscale" {
		t.Errorf("expected upscale, got %s", domain.ModeUpscale)
	}
}

func TestImageSpec_ApplyDefaults(t *testing.T) {
	// Default Text2Img
	spec := &domain.ImageSpec{
		RawPrompt: "a scenic forest",
	}
	spec.ApplyDefaults()
	if spec.Mode != domain.ModeText2Img {
		t.Errorf("expected ModeText2Img, got %s", spec.Mode)
	}
	if spec.DenoiseStrength != 0.0 {
		t.Errorf("expected 0.0 denoise for text2img, got %f", spec.DenoiseStrength)
	}

	// Img2Img with default denoise
	specImg := &domain.ImageSpec{
		RawPrompt:      "transform into cyber city",
		InputImagePath: "base.png",
	}
	specImg.ApplyDefaults()
	if specImg.Mode != domain.ModeImg2Img {
		t.Errorf("expected ModeImg2Img, got %s", specImg.Mode)
	}
	if specImg.DenoiseStrength != 0.70 {
		t.Errorf("expected default 0.70 denoise for img2img, got %f", specImg.DenoiseStrength)
	}

	// Inpaint default mode detection
	specInpaint := &domain.ImageSpec{
		RawPrompt:      "remove background",
		InputImagePath: "base.png",
		MaskImagePath:  "mask.png",
	}
	specInpaint.ApplyDefaults()
	if specInpaint.Mode != domain.ModeInpaint {
		t.Errorf("expected ModeInpaint, got %s", specInpaint.Mode)
	}
	if specInpaint.DenoiseStrength != 0.70 {
		t.Errorf("expected default 0.70 denoise for inpaint, got %f", specInpaint.DenoiseStrength)
	}

	// Preserved explicit denoise strength
	specExplicit := &domain.ImageSpec{
		RawPrompt:       "stylize",
		Mode:            domain.ModeStyleTransfer,
		InputImagePath:  "base.png",
		DenoiseStrength: 0.45,
	}
	specExplicit.ApplyDefaults()
	if specExplicit.DenoiseStrength != 0.45 {
		t.Errorf("expected 0.45 denoise, got %f", specExplicit.DenoiseStrength)
	}

	// Clamping out-of-bounds denoise
	specOver := &domain.ImageSpec{
		Mode:            domain.ModeImg2Img,
		DenoiseStrength: 1.5,
	}
	specOver.ApplyDefaults()
	if specOver.DenoiseStrength != 1.0 {
		t.Errorf("expected clamped 1.0 denoise, got %f", specOver.DenoiseStrength)
	}

	specUnder := &domain.ImageSpec{
		Mode:            domain.ModeImg2Img,
		DenoiseStrength: -0.2,
	}
	specUnder.ApplyDefaults()
	if specUnder.DenoiseStrength != 0.0 {
		t.Errorf("expected clamped 0.0 denoise, got %f", specUnder.DenoiseStrength)
	}
}

func TestImageSpec_Helpers(t *testing.T) {
	specT2I := &domain.ImageSpec{Mode: domain.ModeText2Img}
	if specT2I.IsImg2Img() || specT2I.IsInpaint() || specT2I.IsUpscale() {
		t.Errorf("text2img should not be img2img, inpaint, or upscale")
	}

	specI2I := &domain.ImageSpec{Mode: domain.ModeImg2Img, InputImagePath: "input.png"}
	if !specI2I.IsImg2Img() {
		t.Errorf("expected IsImg2Img to be true")
	}
	if specI2I.IsInpaint() || specI2I.IsUpscale() {
		t.Errorf("expected IsInpaint and IsUpscale to be false")
	}

	specInp := &domain.ImageSpec{Mode: domain.ModeInpaint, InputImagePath: "input.png", MaskImagePath: "mask.png"}
	if !specInp.IsInpaint() {
		t.Errorf("expected IsInpaint to be true")
	}

	specStyle := &domain.ImageSpec{Mode: domain.ModeStyleTransfer, InputImagePath: "input.png"}
	if !specStyle.IsImg2Img() {
		t.Errorf("expected style transfer to be img2img")
	}

	specUpscaleMode := &domain.ImageSpec{Mode: domain.ModeUpscale}
	if !specUpscaleMode.IsUpscale() {
		t.Errorf("expected ModeUpscale to be upscale")
	}

	specUpscaleScale := &domain.ImageSpec{ScaleFactor: 2}
	if !specUpscaleScale.IsUpscale() {
		t.Errorf("expected ScaleFactor > 1 to be upscale")
	}

	specUpscaleFace := &domain.ImageSpec{RestoreFaces: true}
	if !specUpscaleFace.IsUpscale() {
		t.Errorf("expected RestoreFaces to be upscale")
	}
}

func TestImageSpec_Upscale_ApplyDefaults(t *testing.T) {
	// ModeUpscale with no scale factor -> default 4, no face restore
	spec := &domain.ImageSpec{
		Mode:           domain.ModeUpscale,
		InputImagePath: "photo.png",
	}
	spec.ApplyDefaults()
	if spec.ScaleFactor != 4 {
		t.Errorf("expected ScaleFactor 4, got %d", spec.ScaleFactor)
	}
	if spec.RestoreFaces {
		t.Errorf("expected RestoreFaces false")
	}
	if spec.FaceFidelity != 0.0 {
		t.Errorf("expected FaceFidelity 0.0 when RestoreFaces is false, got %f", spec.FaceFidelity)
	}

	// RestoreFaces true without explicit fidelity -> default 0.75 and ModeUpscale
	specFaces := &domain.ImageSpec{
		InputImagePath: "portrait.png",
		RestoreFaces:   true,
	}
	specFaces.ApplyDefaults()
	if specFaces.Mode != domain.ModeUpscale {
		t.Errorf("expected ModeUpscale when RestoreFaces is true, got %s", specFaces.Mode)
	}
	if specFaces.ScaleFactor != 4 {
		t.Errorf("expected ScaleFactor 4 default, got %d", specFaces.ScaleFactor)
	}
	if specFaces.FaceFidelity != 0.75 {
		t.Errorf("expected FaceFidelity 0.75 default, got %f", specFaces.FaceFidelity)
	}

	// RestoreFaces true with explicit fidelity within range
	specFidelity := &domain.ImageSpec{
		Mode:         domain.ModeUpscale,
		RestoreFaces: true,
		FaceFidelity: 0.85,
	}
	specFidelity.ApplyDefaults()
	if specFidelity.FaceFidelity != 0.85 {
		t.Errorf("expected FaceFidelity 0.85, got %f", specFidelity.FaceFidelity)
	}

	// Clamp fidelity out of bounds
	specClampedHigh := &domain.ImageSpec{
		Mode:         domain.ModeUpscale,
		RestoreFaces: true,
		FaceFidelity: 1.5,
	}
	specClampedHigh.ApplyDefaults()
	if specClampedHigh.FaceFidelity != 1.0 {
		t.Errorf("expected clamped FaceFidelity 1.0, got %f", specClampedHigh.FaceFidelity)
	}

	specClampedLow := &domain.ImageSpec{
		Mode:         domain.ModeUpscale,
		RestoreFaces: true,
		FaceFidelity: -0.5,
	}
	specClampedLow.ApplyDefaults()
	if specClampedLow.FaceFidelity != 0.0 {
		t.Errorf("expected clamped FaceFidelity 0.0, got %f", specClampedLow.FaceFidelity)
	}
}

func TestImageSpec_Upscale_Validate(t *testing.T) {
	tests := []struct {
		name        string
		spec        domain.ImageSpec
		expectError bool
	}{
		{
			name: "valid scale 2",
			spec: domain.ImageSpec{
				Mode:           domain.ModeUpscale,
				InputImagePath: "image.png",
				ScaleFactor:    2,
			},
			expectError: false,
		},
		{
			name: "valid scale 4",
			spec: domain.ImageSpec{
				Mode:           domain.ModeUpscale,
				InputImagePath: "image.png",
				ScaleFactor:    4,
			},
			expectError: false,
		},
		{
			name: "valid scale 8",
			spec: domain.ImageSpec{
				Mode:           domain.ModeUpscale,
				InputImagePath: "image.png",
				ScaleFactor:    8,
			},
			expectError: false,
		},
		{
			name: "invalid scale 3",
			spec: domain.ImageSpec{
				Mode:           domain.ModeUpscale,
				InputImagePath: "image.png",
				ScaleFactor:    3,
			},
			expectError: true,
		},
		{
			name: "invalid scale 5",
			spec: domain.ImageSpec{
				Mode:           domain.ModeUpscale,
				InputImagePath: "image.png",
				ScaleFactor:    5,
			},
			expectError: true,
		},
		{
			name: "invalid scale 0",
			spec: domain.ImageSpec{
				Mode:           domain.ModeUpscale,
				InputImagePath: "image.png",
				ScaleFactor:    0,
			},
			expectError: true,
		},
		{
			name: "invalid scale 16",
			spec: domain.ImageSpec{
				Mode:           domain.ModeUpscale,
				InputImagePath: "image.png",
				ScaleFactor:    16,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.spec.Validate()
			if tc.expectError && err == nil {
				t.Errorf("expected validation error for %+v, got nil", tc.spec)
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestImageSpec_Validate(t *testing.T) {
	// Mask provided without input image -> error
	specInvalidMask := &domain.ImageSpec{
		MaskImagePath: "mask.png",
	}
	if err := specInvalidMask.Validate(); err == nil {
		t.Errorf("expected error when mask provided without input image")
	}

	// Valid inpaint spec
	specValidInpaint := &domain.ImageSpec{
		InputImagePath: "base.png",
		MaskImagePath:  "mask.png",
	}
	if err := specValidInpaint.Validate(); err != nil {
		t.Errorf("unexpected error for valid inpaint: %v", err)
	}

	// Valid text2img
	specValidT2I := &domain.ImageSpec{}
	if err := specValidT2I.Validate(); err != nil {
		t.Errorf("unexpected error for valid text2img: %v", err)
	}
}
