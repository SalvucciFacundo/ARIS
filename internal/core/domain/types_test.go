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
	if specT2I.IsImg2Img() || specT2I.IsInpaint() {
		t.Errorf("text2img should not be img2img or inpaint")
	}

	specI2I := &domain.ImageSpec{Mode: domain.ModeImg2Img, InputImagePath: "input.png"}
	if !specI2I.IsImg2Img() {
		t.Errorf("expected IsImg2Img to be true")
	}
	if specI2I.IsInpaint() {
		t.Errorf("expected IsInpaint to be false")
	}

	specInp := &domain.ImageSpec{Mode: domain.ModeInpaint, InputImagePath: "input.png", MaskImagePath: "mask.png"}
	if !specInp.IsInpaint() {
		t.Errorf("expected IsInpaint to be true")
	}

	specStyle := &domain.ImageSpec{Mode: domain.ModeStyleTransfer, InputImagePath: "input.png"}
	if !specStyle.IsImg2Img() {
		t.Errorf("expected style transfer to be img2img")
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
