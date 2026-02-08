package tier2

import (
	"image"
)

// FrequencySplitter decomposes an image into Y/Cb/Cr channels at different resolutions.
// Architecture doc Section 5.4
//
// The eye resolves fine luminance detail but not fine color detail.
// Characters encode luminance patterns; fg/bg colors encode chrominance.
type FrequencySplitter struct {
	// Luminance at full sub-cell resolution
	Y []float64

	// Chrominance at cell resolution (one value per terminal cell)
	Cb []float64
	Cr []float64
}

// Split decomposes the framebuffer into frequency channels.
// Y at full sub-cell resolution, Cb/Cr averaged per cell.
// Uses BT.601 luma coefficients: Y = 0.299R + 0.587G + 0.114B
func (fs *FrequencySplitter) Split(img *image.NRGBA, cellW, cellH int) {
	bounds := img.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()
	termCols := imgW / cellW
	termRows := imgH / cellH

	// Ensure buffers are allocated
	subPixelCount := imgW * imgH
	cellCount := termCols * termRows

	if len(fs.Y) < subPixelCount {
		fs.Y = make([]float64, subPixelCount)
	}
	if len(fs.Cb) < cellCount {
		fs.Cb = make([]float64, cellCount)
		fs.Cr = make([]float64, cellCount)
	}

	// Process each terminal cell
	for ty := 0; ty < termRows; ty++ {
		for tx := 0; tx < termCols; tx++ {
			cellIdx := ty*termCols + tx

			// Accumulators for cell-average chrominance
			var sumCb, sumCr float64
			n := 0

			// Process each sub-pixel within this cell
			for sy := 0; sy < cellH; sy++ {
				for sx := 0; sx < cellW; sx++ {
					px := bounds.Min.X + tx*cellW + sx
					py := bounds.Min.Y + ty*cellH + sy

					// Bounds check
					if px >= bounds.Max.X || py >= bounds.Max.Y {
						continue
					}

					c := img.NRGBAAt(px, py)
					rf := float64(c.R) / 255.0
					gf := float64(c.G) / 255.0
					bf := float64(c.B) / 255.0

					// BT.601 luma coefficients
					y := 0.299*rf + 0.587*gf + 0.114*bf
					cb := bf - y
					cr := rf - y

					// Store luminance at full sub-cell resolution
					subIdx := py*imgW + px
					if subIdx < len(fs.Y) {
						fs.Y[subIdx] = y
					}

					sumCb += cb
					sumCr += cr
					n++
				}
			}

			// Store chrominance at cell resolution (averaged)
			if n > 0 {
				fs.Cb[cellIdx] = sumCb / float64(n)
				fs.Cr[cellIdx] = sumCr / float64(n)
			}
		}
	}
}
