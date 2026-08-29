package imgutil

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
)

// RenderANSIHalfBlocks reads an image file and returns a 24-bit TrueColor ANSI halfblock string.
func RenderANSIHalfBlocks(filePath string, maxCols, maxRows int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	return RenderImageANSI(img, maxCols, maxRows), nil
}

// RenderImageANSI renders an image.Image into ANSI TrueColor halfblocks.
func RenderImageANSI(img image.Image, maxCols, maxRows int) string {
	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	if imgW == 0 || imgH == 0 {
		return ""
	}

	if maxCols <= 0 {
		maxCols = 40
	}
	if maxRows <= 0 {
		maxRows = 20
	}

	// Calculate aspect ratio preserving target dimensions
	// Each terminal cell has roughly a 1:2 width:height character ratio,
	// and half-blocks represent 2 vertical pixels per cell.
	targetCols := maxCols
	targetRows := maxRows * 2 // actual pixel rows

	scaleX := float64(imgW) / float64(targetCols)
	scaleY := float64(imgH) / float64(targetRows)
	scale := scaleX
	if scaleY > scale {
		scale = scaleY
	}
	if scale < 1.0 {
		scale = 1.0
	}

	outW := int(float64(imgW) / scale)
	outH := int(float64(imgH) / scale)
	if outW <= 0 {
		outW = 1
	}
	if outH <= 0 {
		outH = 1
	}

	var buf strings.Builder

	for y := 0; y < outH; y += 2 {
		for x := 0; x < outW; x++ {
			srcX := int(float64(x) * scale)
			srcYTop := int(float64(y) * scale)
			srcYBot := int(float64(y+1) * scale)

			if srcX >= imgW {
				srcX = imgW - 1
			}
			if srcYTop >= imgH {
				srcYTop = imgH - 1
			}

			r1, g1, b1, a1 := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcYTop).RGBA()
			topR, topG, topB := uint8(r1>>8), uint8(g1>>8), uint8(b1>>8)

			if y+1 < outH && srcYBot < imgH {
				r2, g2, b2, a2 := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcYBot).RGBA()
				botR, botG, botB := uint8(r2>>8), uint8(g2>>8), uint8(b2>>8)

				if a1 == 0 && a2 == 0 {
					buf.WriteString(" ")
				} else if a2 == 0 {
					buf.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm▀\x1b[0m", topR, topG, topB))
				} else {
					buf.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
						topR, topG, topB, botR, botG, botB))
				}
			} else {
				if a1 == 0 {
					buf.WriteString(" ")
				} else {
					buf.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm▀\x1b[0m", topR, topG, topB))
				}
			}
		}
		buf.WriteString("\n")
	}

	return strings.TrimRight(buf.String(), "\n")
}
