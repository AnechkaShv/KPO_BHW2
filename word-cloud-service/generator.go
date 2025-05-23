package main

import (
	"bytes"
	"image/png"
	"math/rand"
	"strings"
	"time"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	"image"
	"image/color"
	"image/draw"
)

type WordCloudGenerator struct {
	wordFreqs map[string]int
}

func NewWordCloudGenerator() *WordCloudGenerator {
	return &WordCloudGenerator{
		wordFreqs: make(map[string]int),
	}
}

func (g *WordCloudGenerator) Generate(text string) ([]byte, error) {
	// Process text to count word frequencies
	words := strings.Fields(text)
	g.wordFreqs = make(map[string]int)

	for _, word := range words {
		// Normalize word (lowercase, remove punctuation)
		word = strings.ToLower(word)
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if word != "" {
			g.wordFreqs[word]++
		}
	}

	// Filter out common words (stop words)
	stopWords := map[string]bool{
		"the": true, "and": true, "a": true, "an": true, "in": true,
		"on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "is": true, "are": true, "was": true, "were": true,
		"it": true, "that": true, "this": true, "be": true, "have": true,
	}

	for word := range g.wordFreqs {
		if stopWords[word] || len(word) < 3 {
			delete(g.wordFreqs, word)
		}
	}

	// Limit to top 50 words
	if len(g.wordFreqs) > 50 {
		// Simple way to limit - in production you'd sort and take top N
		for word := range g.wordFreqs {
			if len(g.wordFreqs) <= 50 {
				break
			}
			delete(g.wordFreqs, word)
		}
	}

	// Generate word cloud image
	img := g.generateImage(800, 600)

	// Encode image to PNG
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *WordCloudGenerator) generateImage(width, height int) *image.RGBA {
	// Create a new image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Load font
	fontData := goregular.TTF
	f, err := truetype.Parse(fontData)
	if err != nil {
		panic(err)
	}

	// Prepare random generator
	rand.Seed(time.Now().UnixNano())

	// Draw words
	for word, freq := range g.wordFreqs {
		// Calculate font size based on frequency
		fontSize := 10 + freq*5
		if fontSize > 72 {
			fontSize = 72
		}

		// Create font face
		face := truetype.NewFace(f, &truetype.Options{
			Size:    float64(fontSize),
			DPI:     72,
			Hinting: font.HintingFull,
		})

		// Measure word
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(randomColor()),
			Face: face,
		}

		// Calculate position (simple random placement)
		x := rand.Intn(width - 100)
		y := rand.Intn(height - 50)

		// Draw word
		d.Dot = fixed.P(x, y)
		d.DrawString(word)
	}

	return img
}

func randomColor() color.Color {
	colors := []color.Color{
		color.RGBA{255, 0, 0, 255},     // Red
		color.RGBA{0, 0, 255, 255},     // Blue
		color.RGBA{0, 128, 0, 255},     // Green
		color.RGBA{128, 0, 128, 255},   // Purple
		color.RGBA{255, 165, 0, 255},   // Orange
		color.RGBA{0, 0, 0, 255},       // Black
		color.RGBA{255, 192, 203, 255}, // Pink
		color.RGBA{165, 42, 42, 255},   // Brown
	}
	return colors[rand.Intn(len(colors))]
}
