package accounting_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hypha-dao/document/docgraph"
	"github.com/k0kubun/go-ansi"
	progressbar "github.com/schollz/progressbar/v3"
)

func pause(t *testing.T, seconds time.Duration, headline, prefix string) {
	if headline != "" {
		t.Log(headline)
	}

	bar := progressbar.NewOptions(100,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(90),
		// progressbar.OptionShowIts(),
		progressbar.OptionSetDescription("[cyan]"+fmt.Sprintf("%20v", prefix)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	chunk := seconds / 100
	for i := 0; i < 100; i++ {
		bar.Add(1)
		time.Sleep(chunk)
	}
	fmt.Println()
	fmt.Println()
}

func StrToContentGroups(data string) ([]docgraph.ContentGroup, error) {
	var tempDoc docgraph.Document
	err := json.Unmarshal([]byte(data), &tempDoc)
	if err != nil {
		return nil, fmt.Errorf("Json unmarshal : %v", err)
	}

	return tempDoc.ContentGroups, nil
}

func GetContent(d *docgraph.Document, label string) (*docgraph.ContentItem, error) {
	for _, contentGroup := range d.ContentGroups {
		for _, content := range contentGroup {
			if content.Label == label {
				return &content, nil
			}
		}
	}
	return nil, nil
}

func ReplaceContent(d *docgraph.Document, label string, newLabel string, value *docgraph.FlexValue) error {
	for _, contentGroup := range d.ContentGroups {
		for i := range contentGroup {
			if contentGroup[i].Label == label {
				contentGroup[i].Label = newLabel
				contentGroup[i].Value = value
				return nil
			}
		}
	}
	return nil
}
