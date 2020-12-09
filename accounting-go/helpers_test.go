package accounting_test

import (
	"fmt"
	"testing"
	"time"
	"encoding/json"
	"github.com/k0kubun/go-ansi"
	"github.com/hypha-dao/document/docgraph"
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