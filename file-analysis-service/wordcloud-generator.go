package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type WordCloudGenerator struct{}

func (g *WordCloudGenerator) Generate(text string) ([]byte, error) {
	resp, err := http.Get(
		fmt.Sprintf("https://quickchart.io/wordcloud?text=%s", url.QueryEscape(text)),
	)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
