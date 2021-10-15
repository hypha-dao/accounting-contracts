package accounting_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"
	"strings"

	// eostest "github.com/digital-scarcity/eos-go-test"
	"github.com/eoscanada/eos-go"
	"github.com/hypha-dao/accounting-go"
	"github.com/hypha-dao/document-graph/docgraph"

	"gotest.tools/assert"
)

// var env *Environment

//var chainResponsePause, votingPause, periodPause time.Duration

// var claimedPeriods uint64.

func CreateTestLedger(env *Environment, t *testing.T) string {

	ledgerCgs, err := StrToContentGroups(ledger_tester)
	assert.NilError(t, err)

	_, err = accounting.AddLedger(env.ctx,
		&env.api,
		env.Accounting,
		env.Accounting,
		ledgerCgs)

	assert.NilError(t, err)

	//TODO: I need a way to get the hash with the content groups in go
	docs, err := docgraph.GetDocumentsWithEdge(env.ctx, &env.api, env.Accounting, env.Root, eos.Name("ledger"))

	assert.NilError(t, err)

	return docs[0].Hash.String()

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

func CreateAccount(t *testing.T, env *Environment, data string, parent, ledger eos.Checksum256) (docgraph.Document, error) {

	accountCgs, err := StrToContentGroups(data)

	if err != nil {
		return docgraph.Document{}, err
	}

	BuildAccount(parent, ledger, accountCgs)

	_, err = accounting.CreateAcct(env.ctx,
		&env.api,
		env.Accounting,
		env.Accounting,
		accountCgs)

	if err != nil {
		return docgraph.Document{}, err
	}

	pause(t, time.Second, "", "")

	doc, err := docgraph.GetLastDocumentOfEdge(env.ctx,
		&env.api,
		env.Accounting, eos.Name("account"))

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

func createTrx (trxComponents []accounting.TrxComponent, ledgerDoc *docgraph.Document) (*docgraph.Document, error) {

	var trxDoc docgraph.Document
	var components = ""

	for i, trxComp := range trxComponents {
		genericTrxComp := generic_trx_component
		genericTrxComp = strings.Replace(genericTrxComp, "component_account", trxComp.AccountHash, 1)
		genericTrxComp = strings.Replace(genericTrxComp, "component_amount", trxComp.Amount.String(), 1)
		genericTrxComp = strings.Replace(genericTrxComp, "component_type", trxComp.Type, 1)

		if i > 0 {
			components = components + ",\n"
		}

		components = components + genericTrxComp
	}

	if len(trxComponents) > 0 {
		components = "," + components
	}

	var trxCgs []docgraph.ContentGroup
	
	trxContentGroups := generic_trx
	trxContentGroups = strings.Replace(trxContentGroups, "generic_trx_components", components, 1)
	trxContentGroups = strings.Replace(trxContentGroups, "trx_ledger_value", ledgerDoc.Hash.String(), 1)

	fmt.Println("Trx:", trxContentGroups)

	trxCgs, err := StrToContentGroups(trxContentGroups)

	if err != nil {
		return nil, fmt.Errorf("error converting content groups for generic transaction %v", err)
	}

	trxDoc.ContentGroups = trxCgs

	return &trxDoc, nil

}

// func initTransaction(t *testing.T, ledgerDoc, expensesAcc, incomeAcc, mktingAcc, trxDoc *docgraph.Document, balanced bool) {

// 	var err error

// 	t.Log("Creating accounts...\n")

// 	*expensesAcc, err = CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)

// 	assert.NilError(t, err)

// 	*incomeAcc, err = CreateAccount(t, env, account_income, ledgerDoc.Hash, ledgerDoc.Hash)

// 	assert.NilError(t, err)

// 	*mktingAcc, err = CreateAccount(t, env, account_mkting, expensesAcc.Hash, ledgerDoc.Hash)

// 	assert.NilError(t, err)

// 	salaryAcc, err := CreateAccount(t, env, account_salary, incomeAcc.Hash, ledgerDoc.Hash)

// 	assert.NilError(t, err)

// 	var trxCgs []docgraph.ContentGroup

// 	if balanced {
// 		trxCgs, err := StrToContentGroups(balanced_trx)
// 	} else {
// 		trxCgs, err := StrToContentGroups(unbalanced_trx)
// 	}

// 	assert.NilError(t, err)

// 	trxDoc.ContentGroups = trxCgs

// 	err = ReplaceContent(trxDoc, "account_a", "account",
// 		&docgraph.FlexValue{
// 			BaseVariant: eos.BaseVariant{
// 				TypeID: docgraph.GetVariants().TypeID("checksum256"),
// 				Impl:   mktingAcc.Hash,
// 			}})

// 	assert.NilError(t, err)

// 	err = ReplaceContent(trxDoc, "account_b", "account", &docgraph.FlexValue{
// 		BaseVariant: eos.BaseVariant{
// 			TypeID: docgraph.GetVariants().TypeID("checksum256"),
// 			Impl:   salaryAcc.Hash,
// 		}})

// 	assert.NilError(t, err)

// 	err = ReplaceContent(trxDoc, "trx_ledger", "trx_ledger", &docgraph.FlexValue{
// 		BaseVariant: eos.BaseVariant{
// 			TypeID: docgraph.GetVariants().TypeID("checksum256"),
// 			Impl:   ledgerDoc.Hash,
// 		}})

// 	assert.NilError(t, err)

// }

/* func TestAddLedgerAction(t *testing.T) {

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

	t.Run("Testing Create Account action", func(t *testing.T) {

		ledgerHashStr := CreateTestLedger(t)

		pause(t, time.Second, "", "")

		ledgerDoc, err := docgraph.LoadDocument(env.ctx,
			&env.api,
			env.Accounting,
			ledgerHashStr)

		assert.NilError(t, err)

		expensesAcc, err := CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		mktingAcc, err := CreateAccount(t, env, account_mkting, expensesAcc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		_, err = CreateAccount(t, env, account_salary, mktingAcc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		//Test error when
		_, err = CreateAccount(t, env, account_mkting, expensesAcc.Hash, ledgerDoc.Hash)

		assert.Assert(t, err != nil)		
	})
} */

// func TestCreateTrx(t *testing.T) {
// 	teardownTestCase := setupTestCase(t)
// 	defer teardownTestCase(t)

// 	// var env Environment
// 	env = SetupEnvironment(t)

// 	var expensesAcc, incomeAcc, mktingAcc docgraph.Document

// 	t.Run("Testings CreateTrx: ", func(t *testing.T) {
// 		t.Log(env.String())
// 		t.Log("\nDAO Environment Setup complete\n")

// 		ledgerHashStr := CreateTestLedger(t)

// 		pause(t, time.Second, "", "")

// 		ledgerDoc, err := docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, ledgerHashStr)

// 		assert.NilError(t, err)

// 		expensesAcc, err = CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)

// 		assert.NilError(t, err)

// 		incomeAcc, err = CreateAccount(t, env, account_income, ledgerDoc.Hash, ledgerDoc.Hash)

// 		assert.NilError(t, err)

// 		mktingAcc, err = CreateAccount(t, env, account_mkting, expensesAcc.Hash, ledgerDoc.Hash)

// 		assert.NilError(t, err)

// 		salaryAcc, err := CreateAccount(t, env, account_salary, incomeAcc.Hash, ledgerDoc.Hash)

// 		usdSymbol, _ := eos.StringToSymbol("2,USD")
// 		husdSymbol, _ := eos.StringToSymbol("2,HUSD")

// 		trxDoc, err := createTrx([]accounting.TrxComponent{
// 			accounting.TrxComponent{
// 				mktingAcc.Hash.String(), 
// 				eos.Asset{ Amount: 100000, Symbol: usdSymbol },
// 				"DEBIT",
// 			},
// 			accounting.TrxComponent{
// 				salaryAcc.Hash.String(), 
// 				eos.Asset{ Amount: 50000, Symbol: husdSymbol},
// 				"CREDIT",
// 			},
// 			accounting.TrxComponent{
// 				incomeAcc.Hash.String(), 
// 				eos.Asset{ Amount: 50000, Symbol: husdSymbol},
// 				"CREDIT",
// 			},
// 		}, &ledgerDoc)

// 		assert.NilError(t, err)
		
// 		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, "2,USD")

// 		assert.NilError(t, err)

// 		_, err = accounting.CreateTrx(env.ctx, &env.api, env.Accounting, env.Accounting, trxDoc.ContentGroups)
		
// 		assert.ErrorContains(t, err, "Currency HUSD is not allowed")

// 		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, "2,HUSD")

// 		assert.NilError(t, err)

		
// 		// TODO: Test event

		
// 		_, err = accounting.CreateTrx(env.ctx, &env.api, env.Accounting, env.Accounting, trxDoc.ContentGroups)

// 		assert.NilError(t, err)

// 		_, err = accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)
	
// 		assert.NilError(t, err)

// 		pause(t, time.Second * 2, "", "")

// 		//TODO: Test updatetrx
// 	})

// }

func CheckAccountBalances(ledger string, account string, balances []string) (bool) {

	accPos := strings.Index(ledger, account)
	accPosEnd := strings.Index(ledger[accPos:], "endl")

	fmt.Println("CHECK ACCOUNT BALANCES:", ledger[accPos:(accPos + accPosEnd)])

	if len(balances) == 0 {
		fmt.Println("Checking for an account with no balances")
		return ledger[accPos:(accPos + accPosEnd)] == (account + ", Balances:  ")
	}

	for _, balance := range balances {
		if !strings.Contains(ledger[accPos:(accPos + accPosEnd)], balance) {
			return false
		}
	}

	return true

}

func CheckTransaction(trxInfo accounting.TrxNodeInfo, trxFields map[string]string, trxEdgesLength map[string]int) (bool) {

	trxDocToString, err := accounting.PrintDocument(trxInfo.TrxNode)
	if err != nil { return false }

	fmt.Println(trxDocToString)

	if len(trxInfo.Edges["from"]) != trxEdgesLength["from"] {
		fmt.Println("TRX NUM EDGES FROM DOESN'T MATCH:", "actual:", len(trxInfo.Edges["from"]), ", expected:", trxEdgesLength["from"])
		return false 
	}
	if len(trxInfo.Edges["to"]) != trxEdgesLength["to"] { 
		fmt.Println("TRX NUM EDGES FROM DOESN'T MATCH:", "actual:", len(trxInfo.Edges["to"]), ", expected:", trxEdgesLength["to"])
		return false 
	}
	
	for key, element := range trxFields {
		if !strings.Contains(trxDocToString, key + " : " + element) {
			return false
		}
	}

	return true
}

func SetupTrxTestInfo(env *Environment, t *testing.T) (accounting.TrxTestInfo) {

	ledgerHashStr := CreateTestLedger(env, t)

	pause(t, time.Second, "", "")

	ledgerDoc, err := docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, ledgerHashStr)
	assert.NilError(t, err)

	expensesAcc, err := CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	incomeAcc, err := CreateAccount(t, env, account_income, ledgerDoc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	mktingAcc, err := CreateAccount(t, env, account_mkting, expensesAcc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	salaryAcc, err := CreateAccount(t, env, account_salary, incomeAcc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	usd2Symbol, _ := eos.StringToSymbol("2,USD")
	husd2Symbol, _ := eos.StringToSymbol("2,HUSD")

	_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, "2,USD")
	assert.NilError(t, err)

	_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, "2,HUSD")
	assert.NilError(t, err)

	return accounting.TrxTestInfo {
		Ledger: ledgerDoc,
		Accounts: map[string]docgraph.Document { 
			"Expenses": expensesAcc,
			"Income": incomeAcc,
			"Marketing": mktingAcc,
			"Salary": salaryAcc,
		},
		Currencies: map[string]eos.Symbol {
			"USD2": usd2Symbol,
			"HUSD2": husd2Symbol,
		},
	}

}

func TestDeletetrx(t *testing.T) {

	t.Run("Test delete unapproved transaction", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger
		salaryAcc := trxInfo.Accounts["Salary"]

		usd2Symbol := trxInfo.Currencies["USD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)
		assert.NilError(t, err)

		trxFromChainDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)
		
		_, err = accounting.Deletetrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, trxFromChainDoc.Hash)
		assert.NilError(t, err)

		_, err = docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.ErrorContains(t, err, "no document with edge found: transaction")

	})

}

func TestUpserttrx(t *testing.T) {

 	t.Run("Test insert without approval", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		expensesAcc := trxInfo.Accounts["Expenses"]
		incomeAcc := trxInfo.Accounts["Income"]
		mktingAcc := trxInfo.Accounts["Marketing"]
		salaryAcc := trxInfo.Accounts["Salary"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
			},
			accounting.TrxComponent{
				expensesAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)
		assert.NilError(t, err)

		trxFromChainDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		trxNodeInfo, err := accounting.GetTrxNodeInfo(env.ctx, &env.api, env.Accounting, trxFromChainDoc)
		assert.NilError(t, err)

		trxFields := make(map[string]string)
		edgesLength := make(map[string]int)

		trxFields["trx_memo"] = "Test transaction"
		trxFields["trx_name"] = "transaction name"
		trxFields["id"] = "1"

		edgesLength["from"] = 5
		edgesLength["to"] = 5

		assert.Assert(t, CheckTransaction(trxNodeInfo, trxFields, edgesLength))	

		fmt.Println("---------------------------------")

		ledgerToString, err := accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)
		
		fmt.Println("---------------------------------")
		fmt.Println(ledgerToString)

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{}))
		assert.Assert(t, CheckAccountBalances(ledgerToString, "Income", []string{}))
		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{}))
		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{}))

		fmt.Print("\n\n\n")

	})

	t.Run("Test insert with approval", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		expensesAcc := trxInfo.Accounts["Expenses"]
		incomeAcc := trxInfo.Accounts["Income"]
		mktingAcc := trxInfo.Accounts["Marketing"]
		salaryAcc := trxInfo.Accounts["Salary"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
			},
			accounting.TrxComponent{
				expensesAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, true)
		assert.NilError(t, err)

		trxFromChainDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		trxNodeInfo, err := accounting.GetTrxNodeInfo(env.ctx, &env.api, env.Accounting, trxFromChainDoc)
		assert.NilError(t, err)

		trxFields := make(map[string]string)
		edgesLength := make(map[string]int)

		trxFields["trx_memo"] = "Test transaction"
		trxFields["trx_name"] = "transaction name"
		trxFields["id"] = "1"

		edgesLength["from"] = 5
		edgesLength["to"] = 5

		assert.Assert(t, CheckTransaction(trxNodeInfo, trxFields, edgesLength))	

		fmt.Println("---------------------------------")

		ledgerToString, err := accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)
		
		fmt.Println("---------------------------------")
		fmt.Println(ledgerToString)

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Income", []string{
			"[account_USD:-1000.00 USD]", "[global_USD:-1000.00 USD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{
			"[account_HUSD:500.00 HUSD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{
			"[account_HUSD:-500.00 HUSD]", "[global_HUSD:-500.00 HUSD]", "[global_USD:1000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[account_USD:1000.00 USD]", "[global_USD:1000.00 USD]",
		}))

		pause(t, time.Second * 2, "", "")


		usd3Symbol, _ := eos.StringToSymbol("3,USD")

		trxDoc2, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd3Symbol },
				"CREDIT",
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 80000, Symbol: usd3Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 20000, Symbol: usd3Symbol },
				"DEBIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc2.ContentGroups, true)
		assert.NilError(t, err)

		trxFromChainDoc2, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		trxNodeInfo2, err := accounting.GetTrxNodeInfo(env.ctx, &env.api, env.Accounting, trxFromChainDoc2)
		assert.NilError(t, err)

		trxFields2 := make(map[string]string)
		edgesLength2 := make(map[string]int)

		trxFields2["trx_memo"] = "Test transaction"
		trxFields2["trx_name"] = "transaction name"
		trxFields2["id"] = "2"

		edgesLength2["from"] = 4
		edgesLength2["to"] = 4

		assert.Assert(t, CheckTransaction(trxNodeInfo2, trxFields2, edgesLength2))	

		fmt.Println("---------------------------------")

		ledgerToString, err = accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)

		fmt.Print("LEDGER:\n", ledgerToString, "\n\n\n")

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Income", []string{
			"[account_USD:-980.000 USD]", "[global_USD:-900.000 USD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{
			"[account_HUSD:500.00 HUSD]", "[global_HUSD:500.00 HUSD]","[account_USD:80.000 USD]", "[global_USD:80.000 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{
			"[account_HUSD:-500.00 HUSD]", "[global_HUSD:-500.00 HUSD]", "[global_USD:900.000 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[account_USD:900.000 USD]", "[global_USD:900.000 USD]",
		}))

		fmt.Print("\n\n\n")

	})

	t.Run("Test update without approval", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		incomeAcc := trxInfo.Accounts["Income"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)
		assert.NilError(t, err)

		trxFromChainDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		trxNodeInfo, err := accounting.GetTrxNodeInfo(env.ctx, &env.api, env.Accounting, trxFromChainDoc)
		assert.NilError(t, err)

		trxFields := make(map[string]string)
		edgesLength := make(map[string]int)

		trxFields["trx_memo"] = "Test transaction"
		trxFields["trx_name"] = "transaction name"
		trxFields["id"] = "1"

		edgesLength["from"] = 3
		edgesLength["to"] = 3

		assert.Assert(t, CheckTransaction(trxNodeInfo, trxFields, edgesLength))	


		fmt.Println("---------------------------------")

		expensesAcc := trxInfo.Accounts["Expenses"]
		salaryAcc := trxInfo.Accounts["Salary"]

		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxUpdatedDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
			},
			accounting.TrxComponent{
				expensesAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, trxFromChainDoc.Hash, trxUpdatedDoc.ContentGroups, false)
		assert.NilError(t, err)

		trxFromChainDoc2, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		trxNodeInfo2, err := accounting.GetTrxNodeInfo(env.ctx, &env.api, env.Accounting, trxFromChainDoc2)
		assert.NilError(t, err)

		trxFields2 := make(map[string]string)
		edgesLength2 := make(map[string]int)

		trxFields2["trx_memo"] = "Test transaction"
		trxFields2["trx_name"] = "transaction name"
		trxFields2["id"] = "1"

		edgesLength2["from"] = 5
		edgesLength2["to"] = 5

		assert.Assert(t, CheckTransaction(trxNodeInfo2, trxFields2, edgesLength2))	

	})

	t.Run("Test update with approval", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		incomeAcc := trxInfo.Accounts["Income"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)
		assert.NilError(t, err)

		trxFromChainDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		trxNodeInfo, err := accounting.GetTrxNodeInfo(env.ctx, &env.api, env.Accounting, trxFromChainDoc)
		assert.NilError(t, err)

		trxFields := make(map[string]string)
		edgesLength := make(map[string]int)

		trxFields["trx_memo"] = "Test transaction"
		trxFields["trx_name"] = "transaction name"
		trxFields["id"] = "1"

		edgesLength["from"] = 3
		edgesLength["to"] = 3

		assert.Assert(t, CheckTransaction(trxNodeInfo, trxFields, edgesLength))	


		fmt.Println("---------------------------------")

		expensesAcc := trxInfo.Accounts["Expenses"]
		salaryAcc := trxInfo.Accounts["Salary"]

		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxUpdatedDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
			},
			accounting.TrxComponent{
				expensesAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, trxFromChainDoc.Hash, trxUpdatedDoc.ContentGroups, true)
		assert.NilError(t, err)

		trxFromChainDoc2, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		trxNodeInfo2, err := accounting.GetTrxNodeInfo(env.ctx, &env.api, env.Accounting, trxFromChainDoc2)
		assert.NilError(t, err)

		trxFields2 := make(map[string]string)
		edgesLength2 := make(map[string]int)

		trxFields2["trx_memo"] = "Test transaction"
		trxFields2["trx_name"] = "transaction name"
		trxFields2["id"] = "1"

		edgesLength2["from"] = 5
		edgesLength2["to"] = 5

		assert.Assert(t, CheckTransaction(trxNodeInfo2, trxFields2, edgesLength2))	
		
		fmt.Println("---------------------------------")
		ledgerToString, err := accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)
		fmt.Println(ledgerToString)

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Income", []string{
			"[account_USD:-1000.00 USD]", "[global_USD:-1000.00 USD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{
			"[account_HUSD:500.00 HUSD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{
			"[account_HUSD:-500.00 HUSD]", "[global_HUSD:-500.00 HUSD]", "[global_USD:1000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[account_USD:1000.00 USD]", "[global_USD:1000.00 USD]",
		}))

		pause(t, time.Second * 2, "", "")

	})

	t.Run("Test failures", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		incomeAcc := trxInfo.Accounts["Income"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		usd3Symbol, _ := eos.StringToSymbol("3,USD")

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 10000000, Symbol: usd3Symbol },
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		fmt.Println("Testing it fails if the transaction is not balanced")

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, true)
		assert.ErrorContains(t, err, "Transaction is unbalanced. Asset USD sums up to -9000.000 USD")

		fmt.Println("Testing it fails if the transaction is already approved")

		trxDoc2, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd3Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd3Symbol },
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc2.ContentGroups, true)
		assert.NilError(t, err)

		trxFromChainDoc2, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		trxDoc3, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 10000, Symbol: usd3Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 10000, Symbol: usd3Symbol },
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, trxFromChainDoc2.Hash, trxDoc3.ContentGroups, true)
		assert.ErrorContains(t, err, "Cannot modify an approved transaction: 89bf86f472cfc53462a0de9084c46b63e156500749e026e026b7d2ae44543579")


		fmt.Println("Testing it fails if the transaction doesn't have at least one component")

		trxDoc4, err := createTrx([]accounting.TrxComponent{}, &ledgerDoc)
		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc4.ContentGroups, false)
		assert.ErrorContains(t, err, "Transaction must contain at least 1 component")

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc4.ContentGroups, true)
		assert.ErrorContains(t, err, "Transaction must contain at least 1 component")

	})

}

/* func TestBalanceTrx(t *testing.T) {

	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	t.Run("Balanced Transaction Test: ", func(t *testing.T) {

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(t)

		t.Log(env.String())
		t.Log("\nDAO Environment Setup complete\n")

		ledgerDoc := trxInfo.Ledger

		expensesAcc := trxInfo.Accounts["Expenses"]
		incomeAcc := trxInfo.Accounts["Income"]
		mktingAcc := trxInfo.Accounts["Marketing"]
		salaryAcc := trxInfo.Accounts["Salary"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
			},
			accounting.TrxComponent{
				expensesAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.CreateTrx(env.ctx, &env.api, env.Accounting, env.Accounting, trxDoc.ContentGroups)
		assert.NilError(t, err)

		pause(t, time.Second * 2, "", "")

		trxFromChainDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		fmt.Println("TRANSACTION HASH:", trxFromChainDoc.Hash)
		
		_, err = accounting.BalanceTrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, trxFromChainDoc.Hash)
		assert.NilError(t, err)

		ledgerToString, err := accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)

		fmt.Print("LEDGER:\n", ledgerToString, "\n\n\n")

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Income", []string{
			"[account_USD:-1000.00 USD]", "[global_USD:-1000.00 USD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{
			"[account_HUSD:500.00 HUSD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{
			"[account_HUSD:-500.00 HUSD]", "[global_HUSD:-500.00 HUSD]", "[global_USD:1000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[account_USD:1000.00 USD]", "[global_USD:1000.00 USD]",
		}))

		pause(t, time.Second * 2, "", "")

		usd3Symbol, _ := eos.StringToSymbol("3,USD")

		trxDoc2, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd3Symbol },
				"CREDIT",
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 80000, Symbol: usd3Symbol },
				"DEBIT",
			},
			accounting.TrxComponent{
				incomeAcc.Hash.String(), 
				eos.Asset{ Amount: 20000, Symbol: usd3Symbol },
				"DEBIT",
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.CreateTrx(env.ctx, &env.api, env.Accounting, env.Accounting, trxDoc2.ContentGroups)
		assert.NilError(t, err)

		pause(t, time.Second * 2, "", "")

		trxFromChainDoc2, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)

		fmt.Println("TRANSACTION HASH:", trxFromChainDoc2.Hash)
		
		_, err = accounting.BalanceTrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, trxFromChainDoc2.Hash)
		assert.NilError(t, err)

		ledgerToString, err = accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)

		fmt.Print("LEDGER:\n", ledgerToString, "\n\n\n")

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Income", []string{
			"[account_USD:-980.000 USD]", "[global_USD:-900.000 USD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{
			"[account_HUSD:500.00 HUSD]", "[global_HUSD:500.00 HUSD]","[account_USD:80.000 USD]", "[global_USD:80.000 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{
			"[account_HUSD:-500.00 HUSD]", "[global_HUSD:-500.00 HUSD]", "[global_USD:900.000 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[account_USD:900.000 USD]", "[global_USD:900.000 USD]",
		}))

	})

}  */

//Test creation of transaction with an event linked to an empty component
/* func TestCreateTrxWe(t *testing.T) {
	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	// var env Environment
	env = SetupEnvironment(t)

	var expensesAcc, incomeAcc docgraph.Document

	t.Run("Testings CreateTrx: ", func(t *testing.T) {
		t.Log(env.String())
		t.Log("\nDAO Environment Setup complete\n")

		ledgerHashStr := CreateTestLedger(t)

		pause(t, time.Second, "", "")

		ledgerDoc, err := docgraph.LoadDocument(env.ctx,
																						&env.api,
																						env.Accounting,
																						ledgerHashStr)

		assert.NilError(t, err)

		expensesAcc, err = CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		incomeAcc, err = CreateAccount(t, env, account_income, ledgerDoc.Hash, ledgerDoc.Hash)

		assert.NilError(t, err)

		//Test trx_1
		eventInfo, err := StrToContentGroups(event_1)

		assert.NilError(t, err)

		_, err = accounting.Event(env.ctx, &env.api, env.Accounting, env.Accounting, eventInfo)

		assert.NilError(t, err)

		eventDoc, err := docgraph.GetLastDocument(env.ctx, &env.api, env.Accounting)

		assert.NilError(t, err)

		trxCgs, err := StrToContentGroups(transaction_test_we)

		assert.NilError(t, err)

		trxDoc := docgraph.Document{}
		trxDoc.ContentGroups = trxCgs
		
		err = ReplaceContent(&trxDoc, "trx_ledger", "trx_ledger", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   ledgerDoc.Hash,
			}})

		assert.NilError(t, err)

		err = ReplaceContent(&trxDoc, "event", "event", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   eventDoc.Hash,
		}})

		assert.NilError(t, err)

		_, err = accounting.CreateTrxWe(env.ctx, &env.api, env.Accounting, env.Accounting, trxDoc.ContentGroups);
		
		assert.NilError(t, err)

		trxRealDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction")) 
		//Fill the transaction with actual components
		trxCgs, err = StrToContentGroups(transaction_test_we_update)

		assert.NilError(t, err)

		trxDoc = docgraph.Document{}
		trxDoc.ContentGroups = trxCgs

		err = ReplaceContent(&trxDoc, "account_a", "account",
			&docgraph.FlexValue{
				BaseVariant: eos.BaseVariant{
					TypeID: docgraph.GetVariants().TypeID("checksum256"),
					Impl:   expensesAcc.Hash,
				}})

		assert.NilError(t, err)

		err = ReplaceContent(&trxDoc, "account_b", "account", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   incomeAcc.Hash,
			}})

		assert.NilError(t, err)

		err = ReplaceContent(&trxDoc, "event", "event", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   eventDoc.Hash,
			}})

		err = ReplaceContent(&trxDoc, "trx_ledger", "trx_ledger", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("checksum256"),
				Impl:   ledgerDoc.Hash,
			}})
			

		assert.NilError(t, err)

		_, err = accounting.UpdateTrx(env.ctx, &env.api, env.Accounting, env.Accounting, trxRealDoc.Hash, trxDoc.ContentGroups);
		
		assert.NilError(t, err)

		_, err = accounting.BalanceTrx(env.ctx, &env.api, env.Accounting, env.Accounting, trxRealDoc.Hash);
		
		assert.NilError(t, err)
		
		pause(t, time.Second * 2, "", "")
	})

	
} */

/* func TestSettings(t *testing.T) {

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

func TestEvent(t *testing.T) {

	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	// var env Environment
	env = SetupEnvironment(t)

	//var ledgerDoc, expensesAcc, incomeAcc, mktingAcc docgraph.Document

	t.Run("Configuring the DAO environment: ", func(t *testing.T) {
		t.Log(env.String())
		t.Log("\nDAO Environment Setup complete\n")
	})

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

	t.Run("Testing events", func(t *testing.T) {

		//Test event
		trxInfo, err := StrToContentGroups(event_1)

		assert.NilError(t, err)

		_, err = accounting.Event(env.ctx, &env.api, env.Accounting, tester, trxInfo)

		assert.NilError(t, err)

		trxDoc, err := docgraph.GetLastDocument(env.ctx, &env.api, env.Accounting)

		assert.NilError(t, err)

		trxSource, err := trxDoc.GetContent("source")

		assert.Equal(t, "btc-treasury-1", trxSource.String())

		trxCursor, err := trxDoc.GetContent("cursor")

		assert.Equal(t, "18a835a0d11c91ab6abdd75bf7df1e67deada952b448193e1d4ad76c6e585dfd;0", trxCursor.String())

		lastCursor, err := accounting.GetCursorFromSource(env.ctx, &env.api, env.Accounting, trxSource.String())

		assert.NilError(t, err)

		assert.Equal(t, lastCursor, trxCursor.String())

		//Must give error since beta is not trusted account
		_, err = accounting.Event(env.ctx, &env.api, env.Accounting, beta, trxInfo)

		assert.Assert(t, err != nil)

		//Test trx_2
		trxInfo, err = StrToContentGroups(event_2)

		assert.NilError(t, err)

		_, err = accounting.Event(env.ctx, &env.api, env.Accounting, tester, trxInfo)

		assert.NilError(t, err)

		trxDoc, err = docgraph.GetLastDocument(env.ctx, &env.api, env.Accounting)

		assert.NilError(t, err)

		trxSource, err = trxDoc.GetContent("source")

		assert.Equal(t, "btc-treasury-2", trxSource.String())

		trxCursor, err = trxDoc.GetContent("cursor")

		assert.Equal(t, "87a835a0d11c91ab6abdd75bf7df1e67deada952b448193e1d4ad76c6e585bbb;9", trxCursor.String())

		lastCursor, err = accounting.GetCursorFromSource(env.ctx, &env.api, env.Accounting, trxSource.String())

		assert.NilError(t, err)

		assert.Equal(t, lastCursor, trxCursor.String())

		//Test trx_2 with different cursor, it should override the trx_2 source
		err = ReplaceContent(&trxDoc, "cursor", "cursor", &docgraph.FlexValue{
			BaseVariant: eos.BaseVariant{
				TypeID: docgraph.GetVariants().TypeID("string"),
				Impl:   ";xabc123_",
			}})

		assert.NilError(t, err)

		_, err = accounting.Event(env.ctx, &env.api, env.Accounting, tester, trxDoc.ContentGroups)

		assert.NilError(t, err)

		//Check with the same source
		lastCursor, err = accounting.GetCursorFromSource(env.ctx, &env.api, env.Accounting, trxSource.String())

		assert.NilError(t, err)

		assert.Equal(t, lastCursor, ";xabc123_")
	})
} */
