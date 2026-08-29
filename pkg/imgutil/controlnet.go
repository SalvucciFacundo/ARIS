package imgutil

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Gaussian 5x5 kernel weights normalized by sum (273).
var gaussianKernel = [5][5]float64{
	{1.0 / 273.0, 4.0 / 273.0, 7.0 / 273.0, 4.0 / 273.0, 1.0 / 273.0},
	{4.0 / 273.0, 16.0 / 273.0, 26.0 / 273.0, 16.0 / 273.0, 4.0 / 273.0},
	{7.0 / 273.0, 26.0 / 273.0, 41.0 / 273.0, 26.0 / 273.0, 7.0 / 273.0},
	{4.0 / 273.0, 16.0 / 273.0, 26.0 / 273.0, 16.0 / 273.0, 4.0 / 273.0},
	{1.0 / 273.0, 4.0 / 273.0, 7.0 / 273.0, 4.0 / 273.0, 1.0 / 273.0},
}

// SavePNG encodes and writes the image in PNG format to the given writer.
func SavePNG(w io.Writer, img image.Image) error {
	return png.Encode(w, img)
}

// PreprocessControlNet dispatches preprocessing based on the ControlNet type.
// For "canny", it runs Canny edge detection. For other types (e.g. depth, openpose), it passes through the image.
func PreprocessControlNet(cnType string, img image.Image) (image.Image, error) {
	if strings.ToLower(strings.TrimSpace(cnType)) == "canny" {
		return PreprocessCanny(img, 100.0, 200.0)
	}
	return img, nil
}

// CannyEdgeDetection is an alias for PreprocessCanny.
func CannyEdgeDetection(img image.Image, lowThresh, highThresh float64) (image.Image, error) {
	return PreprocessCanny(img, lowThresh, highThresh)
}

// PreprocessCanny runs a pure-Go 5-stage Canny edge detection pipeline on the input image.
func PreprocessCanny(img image.Image, lowThresh, highThresh float64) (image.Image, error) {
	if img == nil {
		return nil, fmt.Errorf("nil image provided")
	}

	if lowThresh == 0 && highThresh == 0 {
		lowThresh, highThresh = 100.0, 200.0
	}
	if lowThresh > highThresh {
		lowThresh, highThresh = highThresh, lowThresh
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return nil, fmt.Errorf("empty image dimensions")
	}

	// 1. Grayscale luminance conversion (Y = 0.299R + 0.587G + 0.114B)
	gray := make([][]float64, h)
	for y := 0; y < h; y++ {
		gray[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// Convert 16-bit color channels to [0, 255]
			rf := float64(r >> 8)
			gf := float64(g >> 8)
			bf := float64(b >> 8)
			gray[y][x] = 0.299*rf + 0.587*gf + 0.114*bf
		}
	}

	// 2. Gaussian Smoothing (5x5 filter)
	blurred := make([][]float64, h)
	for y := 0; y < h; y++ {
		blurred[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			var sum float64
			for ky := -2; ky <= 2; ky++ {
				py := y + ky
				if py < 0 {
					py = 0
				} else if py >= h {
					py = h - 1
				}
				for kx := -2; kx <= 2; kx++ {
					px := x + kx
					if px < 0 {
						px = 0
					} else if px >= w {
						px = w - 1
					}
					sum += gray[py][px] * gaussianKernel[ky+2][kx+2]
				}
			}
			blurred[y][x] = sum
		}
	}

	// 3. Sobel Gradient Operators
	mag := make([][]float64, h)
	theta := make([][]float64, h)
	for y := 0; y < h; y++ {
		mag[y] = make([]float64, w)
		theta[y] = make([]float64, w)
	}

	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			gx := -1.0*blurred[y-1][x-1] + 1.0*blurred[y-1][x+1] +
				-2.0*blurred[y][x-1] + 2.0*blurred[y][x+1] +
				-1.0*blurred[y+1][x-1] + 1.0*blurred[y+1][x+1]

			gy := -1.0*blurred[y-1][x-1] - 2.0*blurred[y-1][x] - 1.0*blurred[y-1][x+1] +
				1.0*blurred[y+1][x-1] + 2.0*blurred[y+1][x] + 1.0*blurred[y+1][x+1]

			m := math.Hypot(gx, gy)
			mag[y][x] = m

			angle := math.Atan2(gy, gx) * 180.0 / math.Pi
			if angle < 0 {
				angle += 180.0
			}
			theta[y][x] = angle
		}
	}

	// 4. Non-Maximum Suppression (NMS)
	nms := make([][]float64, h)
	for y := 0; y < h; y++ {
		nms[y] = make([]float64, w)
	}

	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			m := mag[y][x]
			if m == 0 {
				continue
			}

			ang := theta[y][x]
			var q, r float64

			// Angle 0° (Horizontal)
			if (ang >= 0 && ang < 22.5) || (ang >= 157.5 && ang <= 180) {
				q = mag[y][x+1]
				r = mag[y][x-1]
			} else if ang >= 22.5 && ang < 67.5 { // Angle 45°
				q = mag[y+1][x-1]
				r = mag[y-1][x+1]
			} else if ang >= 67.5 && ang < 112.5 { // Angle 90° (Vertical)
				q = mag[y+1][x]
				r = mag[y-1][x]
			} else if ang >= 112.5 && ang < 157.5 { // Angle 135°
				q = mag[y-1][x-1]
				r = mag[y+1][x+1]
			}

			if m >= q && m >= r {
				nms[y][x] = m
			} else {
				nms[y][x] = 0
			}
		}
	}

	// 5. Double Thresholding & Hysteresis Edge Tracking
	out := image.NewGray(image.Rect(0, 0, w, h))

	type point struct{ x, y int }
	var queue []point

	state := make([][]uint8, h) // 0: non-edge, 1: weak, 2: strong
	for y := 0; y < h; y++ {
		state[y] = make([]uint8, w)
		for x := 0; x < w; x++ {
			val := nms[y][x]
			if val >= highThresh {
				state[y][x] = 2
				queue = append(queue, point{x, y})
			} else if val >= lowThresh {
				state[y][x] = 1
			}
		}
	}

	// BFS to link weak edges to strong edges
	for len(queue) > 0 {
		pt := queue[0]
		queue = queue[1:]

		for dy := -1; dy <= 1; dy++ {
			ny := pt.y + dy
			if ny < 0 || ny >= h {
				continue
			}
			for dx := -1; dx <= 1; dx++ {
				nx := pt.x + dx
				if nx < 0 || nx >= w || (dx == 0 && dy == 0) {
					continue
				}
				if state[ny][nx] == 1 {
					state[ny][nx] = 2
					queue = append(queue, point{nx, ny})
				}
			}
		}
	}

	// Set output image: strong edges = 255 (white), everything else = 0 (black)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if state[y][x] == 2 {
				out.SetGray(x, y, color.Gray{Y: 255})
			} else {
				out.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}

	return out, nil
}

// PreprocessCannyFile loads an image from inputPath, computes the Canny edge map, and writes it to outputPath.
func PreprocessCannyFile(inputPath, outputPath string, lowThresh, highThresh float64) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input image file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("decode input image: %w", err)
	}

	edges, err := PreprocessCanny(img, lowThresh, highThresh)
	if err != nil {
		return fmt.Errorf("canny edge detection: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	if err := SavePNG(outFile, edges); err != nil {
		return fmt.Errorf("save edge PNG: %w", err)
	}

	return nil
}
