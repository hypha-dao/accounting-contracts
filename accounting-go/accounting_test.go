package accounting_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	eostest "github.com/digital-scarcity/eos-go-test"
	"github.com/eoscanada/eos-go"
	"github.com/hypha-dao/accounting-go"
	"github.com/hypha-dao/document/docgraph"

	"gotest.tools/assert"
)

var env *Environment

//var chainResponsePause, votingPause, periodPause time.Duration

// var claimedPeriods uint64.

func CreateTestLedger(t *testing.T) string {

	ledgerCgs, err := StrToContentGroups(ledger_tester)

	assert.NilError(t, err)

	_, err = accounting.AddLedger(env.ctx,
		&env.api,
		env.Accounting,
		eos.AccountName("testcreate"),
		ledgerCgs)

	assert.NilError(t, err)

	//TODO: I need a way to get the hash with the content groups in go
	return "4c807227a2c9d7ebe5b22050f6d3f0d4318fcb57904e19e18746ae0309024481"
}

func BuildAccount(parent, ledger eos.Checksum256, accountCgs []docgraph.ContentGroup) {

	accountCgs[0] = append(accountCgs[0], docgraph.ContentItem{
		Label: "parent_account",
		Value: &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   parent,
			}},
	})

	accountCgs[0] = append(accountCgs[0], docgraph.ContentItem{
		Label: "ledger_account",
		Value: &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   ledger,
			}},
	})
}

func CreateAccount(env *Environment, data string, parent, ledger eos.Checksum256) (docgraph.Document, error) {

	accountCgs, err := StrToContentGroups(data)

	if err != nil {
		return docgraph.Document{}, err
	}

	BuildAccount(parent, ledger, accountCgs)

	_, err = accounting.CreateAcct(env.ctx,
		&env.api,
		env.Accounting,
		eos.AccountName("tester"),
		accountCgs)

	if err != nil {
		return docgraph.Document{}, err
	}

	doc, err := docgraph.GetLastDocument(env.ctx,
		&env.api,
		env.Accounting)

	return doc, err
}

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

func transactTest(t *testing.T, ledgerDoc, expensesAcc, incomeAcc, mktingAcc *docgraph.Document) {

	var err error

	t.Run(("Testing transact action"), func(t *testing.T) {

		ledgerHashStr := CreateTestLedger(t)

		pause(t, time.Second, "", "")

		*ledgerDoc, err = docgraph.LoadDocument(env.ctx,
			&env.api,
			env.Accounting,
			ledgerHashStr)

		assert.NilError(t, err)

		t.Log("Creating accounts...\n")

		*expensesAcc, err = CreateAccount(env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		*incomeAcc, err = CreateAccount(env, account_income, ledgerDoc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		*mktingAcc, err = CreateAccount(env, account_mkting, expensesAcc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		salaryAcc, err := CreateAccount(env, account_salary, incomeAcc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		trxCgs, err := StrToContentGroups(transaction_test_1)

		assert.NilError(t, err)

		trxDoc := docgraph.Document{}
		trxDoc.ContentGroups = trxCgs

		err = ReplaceContent(&trxDoc, "account_a", "account",
			&docgraph.FlexValue{
				BaseVariant: eos.BaseVariant{
					TypeID: docgraph.GetVariants().TypeID("checksum256"),
					Impl:   mktingAcc.Hash,
				}})

		assert.NilError(t, err)

		err = ReplaceContent(&trxDoc, "account_b", "account", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   salaryAcc.Hash,
			}})

		assert.NilError(t, err)

		err = ReplaceContent(&trxDoc, "trx_ledger", "trx_ledger", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   ledgerDoc.Hash,
			}})

		assert.NilError(t, err)

		_, err = accounting.Transact(env.ctx,
			&env.api,
			env.Accounting,
			env.Accounting,
			trxDoc.ContentGroups)

		assert.NilError(t, err)
	})
}

func impliedTransacTest(t *testing.T, ledgerDoc, expensesAcc, incomeAcc, mktingAcc docgraph.Document) {

	t.Run(("Testing implied transaction"), func(t *testing.T) {

		t.Log("Creating food account...\n")

		foodAcc, err := CreateAccount(env, account_food, expensesAcc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		trxCgs, err := StrToContentGroups(transaction_test_implied)

		assert.NilError(t, err)

		trxDoc := docgraph.Document{}
		trxDoc.ContentGroups = trxCgs

		err = ReplaceContent(&trxDoc, "account_a", "account",
			&docgraph.FlexValue{
				BaseVariant: eos.BaseVariant{
					TypeID: docgraph.GetVariants().TypeID("checksum256"),
					Impl:   mktingAcc.Hash,
				}})

		assert.NilError(t, err)

		err = ReplaceContent(&trxDoc, "account_b", "account", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   foodAcc.Hash,
			}})

		assert.NilError(t, err)

		err = ReplaceContent(&trxDoc, "account_c", "account", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   incomeAcc.Hash,
			}})

		assert.NilError(t, err)

		err = ReplaceContent(&trxDoc, "trx_ledger", "trx_ledger", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   ledgerDoc.Hash,
			}})

		assert.NilError(t, err)

		_, err = accounting.Transact(env.ctx,
			&env.api,
			env.Accounting,
			env.Accounting,
			trxDoc.ContentGroups)

		assert.NilError(t, err)
	})
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

		CreateTestLedger(t)

		pause(t, time.Second, "", "")
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

		ledgerHashStr := CreateTestLedger(t)

		pause(t, time.Second, "", "")

		ledgerDoc, err := docgraph.LoadDocument(env.ctx,
			&env.api,
			env.Accounting,
			ledgerHashStr)

		assert.NilError(t, err)

		accountCgs, err := StrToContentGroups(account_mkting)

		assert.NilError(t, err)

		BuildAccount(ledgerDoc.Hash, ledgerDoc.Hash, accountCgs)

		t.Log("Creating simple account...\n")
		_, err = accounting.CreateAcct(env.ctx,
			&env.api,
			env.Accounting,
			eos.AccountName("testcreate"),
			accountCgs)

		assert.NilError(t, err)

		pause(t, time.Second, "", "")

		accountCgs, err = StrToContentGroups(account_openings_tester)

		assert.NilError(t, err)

		BuildAccount(ledgerDoc.Hash, ledgerDoc.Hash, accountCgs)

		t.Log("Creating account with opening balances...\n")
		_, err = accounting.CreateAcct(env.ctx,
			&env.api,
			env.Accounting,
			eos.AccountName("testcreate"),
			accountCgs)

		assert.NilError(t, err)

		pause(t, time.Second, "", "")

		//Test error when
		t.Log("Testing duplicated account...\n")
		_, err = accounting.CreateAcct(env.ctx,
			&env.api,
			env.Accounting,
			eos.AccountName("testcreate"),
			accountCgs)

		assert.Assert(t, err != nil)

		pause(t, time.Second, "", "")

		//accounting.SayHi(env.ctx, &env.api, env.Accounting);
	})
}

func TestTransact(t *testing.T) {
	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	// var env Environment
	env = SetupEnvironment(t)

	var ledgerDoc, expensesAcc, incomeAcc, mktingAcc docgraph.Document

	t.Run("Configuring the DAO environment: ", func(t *testing.T) {
		t.Log(env.String())
		t.Log("\nDAO Environment Setup complete\n")
	})

	transactTest(t, &ledgerDoc, &expensesAcc, &incomeAcc, &mktingAcc)

	impliedTransacTest(t, ledgerDoc, expensesAcc, incomeAcc, mktingAcc)
}

func TestSettings(t *testing.T) {

	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	// var env Environment
	env = SetupEnvironment(t)

	var ledgerDoc, expensesAcc, incomeAcc, mktingAcc docgraph.Document

	t.Run("Configuring the DAO environment: ", func(t *testing.T) {
		t.Log(env.String())
		t.Log("\nDAO Environment Setup complete\n")
	})

	transactTest(t, &ledgerDoc, &expensesAcc, &incomeAcc, &mktingAcc)

	impliedTransacTest(t, ledgerDoc, expensesAcc, incomeAcc, mktingAcc)

	t.Run("Testing Settings", func(t *testing.T) {

		_, err := accounting.SetSetting(env.ctx,
			&env.api,
			env.Accounting,
			"test",
			docgraph.FlexValue{
				BaseVariant: eos.BaseVariant{
					TypeID: docgraph.GetVariants().TypeID("string"),
					Impl:   "test_value",
				}})

		assert.NilError(t, err)

		_, err = accounting.RemSetting(env.ctx,
			&env.api,
			env.Accounting,
			"test")

		assert.NilError(t, err)

		//Create new setting
		_, err = accounting.SetSetting(env.ctx,
			&env.api,
			env.Accounting,
			"test2",
			docgraph.FlexValue{
				BaseVariant: eos.BaseVariant{
					TypeID: docgraph.GetVariants().TypeID("string"),
					Impl:   "test_new",
				}})

		assert.NilError(t, err)

		//Update value
		_, err = accounting.SetSetting(env.ctx,
			&env.api,
			env.Accounting,
			"test2",
			docgraph.FlexValue{
				BaseVariant: eos.BaseVariant{
					TypeID: docgraph.GetVariants().TypeID("string"),
					Impl:   "test_updated",
				}})

		assert.NilError(t, err)
	})
}

func TestUnreviewedTransaction(t *testing.T) {

	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	// var env Environment
	env = SetupEnvironment(t)

	//var ledgerDoc, expensesAcc, incomeAcc, mktingAcc docgraph.Document

	t.Run("Configuring the DAO environment: ", func(t *testing.T) {
		t.Log(env.String())
		t.Log("\nDAO Environment Setup complete\n")
	})

	// transactTest(t, &ledgerDoc, &expensesAcc, &incomeAcc, &mktingAcc)

	// impliedTransacTest(t, ledgerDoc, expensesAcc, incomeAcc, mktingAcc)

	_, tester, _ := eostest.CreateAccountWithRandomKey(env.ctx, &env.api, "tester")
	_, alpha, _ := eostest.CreateAccountWithRandomKey(env.ctx, &env.api, "alpha")
	_, beta, _ := eostest.CreateAccountWithRandomKey(env.ctx, &env.api, "beta")
	_, gamma, _ := eostest.CreateAccountWithRandomKey(env.ctx, &env.api, "gamma")

	t.Run("Testing trusted accounts", func(t *testing.T) {
		_, err := accounting.AddTrustedAccount(env.ctx, &env.api, env.Accounting, tester)

		assert.NilError(t, err)

		_, err = accounting.AddTrustedAccount(env.ctx, &env.api, env.Accounting, alpha)

		assert.NilError(t, err)

		_, err = accounting.AddTrustedAccount(env.ctx, &env.api, env.Accounting, beta)

		assert.NilError(t, err)

		_, err = accounting.AddTrustedAccount(env.ctx, &env.api, env.Accounting, gamma)

		assert.NilError(t, err)

		_, err = accounting.RemTrustedAccount(env.ctx, &env.api, env.Accounting, beta)

		assert.NilError(t, err)
	})

	t.Run("Testing unreviewed transactions", func(t *testing.T) {

		trxInfo, err := StrToContentGroups(unreviewd_trx_1)

		assert.NilError(t, err)

		_, err = accounting.UnreviewedTrx(env.ctx, &env.api, env.Accounting, tester, trxInfo)

		assert.NilError(t, err)

		//Must give error since beta is not trusted account
		_, err = accounting.UnreviewedTrx(env.ctx, &env.api, env.Accounting, beta, trxInfo)

		assert.Assert(t, err != nil)
	})
}
