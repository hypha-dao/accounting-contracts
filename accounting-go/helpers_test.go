package accounting_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	eostest "github.com/digital-scarcity/eos-go-test"
	eos "github.com/eoscanada/eos-go"
	"github.com/hypha-dao/document-graph/docgraph"
	"github.com/k0kubun/go-ansi"
	progressbar "github.com/schollz/progressbar/v3"
)

type createRoot struct {
	Notes string `json:"notes"`
}

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

func CreateRoot(ctx context.Context, api *eos.API, contract, creator eos.AccountName) (docgraph.Document, error) {
	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("createroot"),
		Authorization: []eos.PermissionLevel{
			{Actor: creator, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(createRoot{
			Notes: "notes",
		}),
	}}
	_, err := eostest.ExecWithRetry(ctx, api, actions)
	if err != nil {
		return docgraph.Document{}, fmt.Errorf("execute create root: %v", err)
	}

	lastDoc, err := docgraph.GetLastDocument(ctx, api, contract)
	if err != nil {
		return docgraph.Document{}, fmt.Errorf("get last document: %v", err)
	}
	return lastDoc, nil
}

func StrToContentGroups(data string) ([]docgraph.ContentGroup, error) {
	var tempDoc docgraph.Document
	err := json.Unmarshal([]byte(data), &tempDoc)
	if err != nil {
		return nil, fmt.Errorf("Json unmarshal : %v", err)
	}

	return tempDoc.ContentGroups, nil
}

func CreateStrContent(label string, value string) {

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
