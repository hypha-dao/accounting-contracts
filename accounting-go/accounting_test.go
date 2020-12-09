package accounting_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/eoscanada/eos-go"
	"github.com/hypha-dao/accounting-go"
	"github.com/hypha-dao/document/docgraph"

	"gotest.tools/assert"
)

var env *Environment

//var chainResponsePause, votingPause, periodPause time.Duration

// var claimedPeriods uint64

func SaveGraph(ctx context.Context, api *eos.API, contract eos.AccountName, folderName string) error {

	var request eos.GetTableRowsRequest
	request.Code = string(contract)
	request.Scope = string(contract)
	request.Table = "documents"
	request.Limit = 1000
	request.JSON = true
	response, err := api.GetTableRows(ctx, request)
	if err != nil {
		return fmt.Errorf("Unable to retrieve rows: %v", err)
	}

	data, err := response.Rows.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Unable to marshal json: %v", err)
	}

	documentsFile := folderName + "/documents.json"
	err = ioutil.WriteFile(documentsFile, data, 0644)
	if err != nil {
		return fmt.Errorf("Unable to write file: %v", err)
	}

	request = eos.GetTableRowsRequest{}
	request.Code = string(contract)
	request.Scope = string(contract)
	request.Table = "edges"
	request.Limit = 1000
	request.JSON = true
	response, err = api.GetTableRows(ctx, request)
	if err != nil {
		return fmt.Errorf("Unable to retrieve rows: %v", err)
	}

	data, err = response.Rows.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Unable to marshal json: %v", err)
	}

	edgesFile := folderName + "/edges.json"
	err = ioutil.WriteFile(edgesFile, data, 0644)
	if err != nil {
		return fmt.Errorf("Unable to write file: %v", err)
	}

	return nil
}

func setupTestCase(t *testing.T) func(t *testing.T) {
	t.Log("Bootstrapping testing environment ...")

	_, err := exec.Command("sh", "-c", "pkill -SIGINT nodeos").Output()
	if err == nil {
		pause(t, time.Second, "Killing nodeos ...", "")
	}

	t.Log("Starting nodeos from 'nodeos.sh' script ...")
	cmd := exec.Command("./nodeos.sh")
	cmd.Stdout = os.Stdout
	err = cmd.Start()
	assert.NilError(t, err)

	t.Log("nodeos PID: ", cmd.Process.Pid)

	pause(t, time.Second, "", "")

	return func(t *testing.T) {

		folderName := "test_results"
		t.Log("Saving graph to : ", folderName)
		os.Mkdir(folderName, 0755)
		//err := SaveGraph(env.ctx, &env.api, env.DAO, folderName)
		//assert.NilError(t, err)
	}
}

func TestAddLedgerAction(t *testing.T) {

	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	// var env Environment
	env = SetupEnvironment(t)

	t.Run("Configuring the DAO environment: ", func(t *testing.T) {
		t.Log(env.String())
		t.Log("\nDAO Environment Setup complete\n")
	})

	t.Run("Testing AddLedger action", func(t *testing.T) {

		ledger, err := StrToContentGroups(ledger_tester)

		assert.NilError(t, err)

		_, err = accounting.AddLedger(env.ctx,
			&env.api,
			env.Accounting,
			eos.AccountName("tester"),
			ledger)

		pause(t, time.Second, "", "")

		assert.NilError(t, err)
		//accounting.SayHi(env.ctx, &env.api, env.Accounting);
	})
}

func TestCreateAccount(t *testing.T) {

	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	// var env Environment
	env = SetupEnvironment(t)

	t.Run("Configuring the DAO environment: ", func(t *testing.T) {
		t.Log(env.String())
		t.Log("\nDAO Environment Setup complete\n")
	})

	t.Run("Testing Create action", func(t *testing.T) {

		ledgerCgs, err := StrToContentGroups(ledger_tester)

		assert.NilError(t, err)

		_, err = accounting.AddLedger(env.ctx,
			&env.api,
			env.Accounting,
			eos.AccountName("testcreate"),
			ledgerCgs)

		assert.NilError(t, err)

		pause(t, time.Second, "", "")

		//TODO: I need a way to get the hash with the content groups in go
		ledgerHashStr := "545df793947527201427c136cb8c817a40c625d350a53b72c141a22e73f85e3b"

		ledgerDoc, err := docgraph.LoadDocument(env.ctx,
			&env.api,
			env.Accounting,
			ledgerHashStr)

		assert.NilError(t, err)

		accountCgs, err := StrToContentGroups(account_tester)

		assert.NilError(t, err)

		accountCgs[0] = append(accountCgs[0], docgraph.ContentItem{
			Label: "parent_account",
			Value: &docgraph.FlexValue{
				BaseVariant: eos.BaseVariant{
					TypeID: docgraph.GetVariants().TypeID("checksum256"),
					Impl:   ledgerDoc.Hash,
				}},
		})

		_, err = accounting.CreateAcct(env.ctx,
			&env.api,
			env.Accounting,
			eos.AccountName("testcreate"),
			accountCgs)

		assert.NilError(t, err)

		pause(t, time.Second, "", "")

		_, err = accounting.CreateAcct(env.ctx,
			&env.api,
			env.Accounting,
			eos.AccountName("testcreate"),
			accountCgs)

		assert.NilError(t, err)

		pause(t, time.Second, "", "")

		//accounting.SayHi(env.ctx, &env.api, env.Accounting);
	})
}
