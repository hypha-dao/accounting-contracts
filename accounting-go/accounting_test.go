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
	"reflect"

	"github.com/eoscanada/eos-go"
	"github.com/hypha-dao/accounting-go"
	"github.com/hypha-dao/document-graph/docgraph"

	"gotest.tools/assert"
)


func CreateTestLedger(env *Environment, t *testing.T) string {

	ledgerCgs, err := StrToContentGroups(ledger_tester)
	assert.NilError(t, err)

	_, err = accounting.AddLedger(env.ctx,
		&env.api,
		env.Accounting,
		env.Accounting,
		ledgerCgs)

	assert.NilError(t, err)

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

		if trxComp.EventHash != nil {
			event := `,{ 
				"label": "event",
				"value": [
					"checksum256",
					"<hash>"
			] }`
			event = strings.Replace(event, "<hash>", trxComp.EventHash.String(), 1)
			genericTrxComp = strings.Replace(genericTrxComp, "<event>", event, 1)
		} else {
			genericTrxComp = strings.Replace(genericTrxComp, "<event>", "", 1)
		}

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

func CreateExchangeRateEntry(from, to string, rate eos.Int64, date eos.TimePoint, t *testing.T) (accounting.ExRateEntry) {

	fromSymbolCode, err := eos.StringToSymbolCode(from)
	assert.NilError(t, err)

	toSymbolCode, err := eos.StringToSymbolCode(to)
	assert.NilError(t, err)

	return accounting.ExRateEntry{
		From: fromSymbolCode,
		To: toSymbolCode,
		Date: date,
		Rate: rate,
	}

}

func CheckAllowedCurrencies(expectedAllowedCurrencies []string, allowedCurrenciesOnChain []eos.Symbol) (bool) {

	fmt.Print("Currencies on chain: ")
	for _, allowedCurrency := range allowedCurrenciesOnChain {
		fmt.Print(allowedCurrency.Symbol, ", ")
	}
	fmt.Print("\n\n")

	if len(expectedAllowedCurrencies) != len(allowedCurrenciesOnChain) {
		fmt.Println("The length of the allowed currencies is not the expected one: current =", len(allowedCurrenciesOnChain), ", expected =", len(expectedAllowedCurrencies))
		return false
	}

	for _, expectedCurrency := range expectedAllowedCurrencies {
		isAllowed := false
		
		for _, allowedOnChain := range allowedCurrenciesOnChain {

			if allowedOnChain.Symbol == expectedCurrency {
				isAllowed = true
				break
			}
	
		}

		if !isAllowed {
			return false
		}
	}

	return true

}

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

	fmt.Println("Ledger HASH:", ledgerDoc.Hash)

	expensesAcc, err := CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	fmt.Println("Expenses HASH:", expensesAcc.Hash)

	incomeAcc, err := CreateAccount(t, env, account_income, ledgerDoc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	fmt.Println("Income HASH:", incomeAcc.Hash)

	mktingAcc, err := CreateAccount(t, env, account_mkting, expensesAcc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	fmt.Println("Marketing HASH:", mktingAcc.Hash)

	developmentAcc, err := CreateAccount(t, env, account_development, expensesAcc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	fmt.Println("Development HASH:", developmentAcc.Hash)

	salaryAcc, err := CreateAccount(t, env, account_salary, incomeAcc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	fmt.Println("Salary HASH:", salaryAcc.Hash)

	salesAcc, err := CreateAccount(t, env, account_sales, incomeAcc.Hash, ledgerDoc.Hash)
	assert.NilError(t, err)

	fmt.Println("Salary HASH:", salesAcc.Hash)


	usd2Symbol, _ := eos.StringToSymbol("2,USD")
	husd2Symbol, _ := eos.StringToSymbol("2,HUSD")

	_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,USD")
	assert.NilError(t, err)

	_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,HUSD")
	assert.NilError(t, err)

	return accounting.TrxTestInfo {
		Ledger: ledgerDoc,
		Accounts: map[string]docgraph.Document { 
			"Expenses": expensesAcc,
			"Income": incomeAcc,
			"Marketing": mktingAcc,
			"Development":developmentAcc,
			"Salary": salaryAcc,
			"Sales": salesAcc,
		},
		Currencies: map[string]eos.Symbol {
			"USD2": usd2Symbol,
			"HUSD2": husd2Symbol,
		},
	}

}



func TestCreateacc(t *testing.T) {

	t.Run("Test create account successfully", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)

		ledgerHashStr := CreateTestLedger(env, t)

		pause(t, time.Second, "", "")
	
		ledgerDoc, err := docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, ledgerHashStr)
		assert.NilError(t, err)
	
		fmt.Println("Ledger HASH:", ledgerDoc.Hash)
	
		_, err = CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)
		assert.NilError(t, err)

	})

	t.Run("Test create tree of accounts", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		developmentAcc := trxInfo.Accounts["Development"]

		fmt.Println("Creating a deep tree")

		i := 0
		deepParent := developmentAcc
		for i < 10 {
			accountData := account_mkting_variant_code
			accountData = strings.Replace(accountData, "<account_code>", fmt.Sprint(i), 1)

			deepAccount, err := CreateAccount(t, env, accountData, deepParent.Hash, ledgerDoc.Hash)
			assert.NilError(t, err)
			
			deepParent = deepAccount
			i += 1
		}

		deepAccountToString, err := accounting.PrintDocument(deepParent)
		assert.NilError(t, err)

		fmt.Println("Deep account:")
		fmt.Println(deepAccountToString)

		assert.Assert(t, strings.Contains(deepAccountToString, "account_code : 9"))

	})

	t.Run("Test only unused accounts can have children", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)

		ledgerHashStr := CreateTestLedger(env, t)

		pause(t, time.Second, "", "")
	
		ledgerDoc, err := docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, ledgerHashStr)
		assert.NilError(t, err)
	
		fmt.Println("Ledger HASH:", ledgerDoc.Hash)
	
		expensesAcc, err := CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)
		assert.NilError(t, err)

		salaryAcc, err := CreateAccount(t, env, account_salary, ledgerDoc.Hash, ledgerDoc.Hash)
		assert.NilError(t, err)

		usd2Symbol, _ := eos.StringToSymbol("2,USD")
	
		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,USD")
		assert.NilError(t, err)
	
		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				expensesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, true)
		assert.NilError(t, err)

		_, err = CreateAccount(t, env, account_development, expensesAcc.Hash, ledgerDoc.Hash)
		assert.ErrorContains(t, err, "Parent account already has associated components. Parent hash: " + expensesAcc.Hash.String())

	}) 

}

func TestUpdateacc(t *testing.T) {

	t.Run("Test update leaf account successfully", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)

		ledgerHashStr := CreateTestLedger(env, t)

		pause(t, time.Second, "", "")
	
		ledgerDoc, err := docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, ledgerHashStr)
		assert.NilError(t, err)
	
		fmt.Println("Ledger HASH:", ledgerDoc.Hash)
	
		expensesAcc, err := CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)
		assert.NilError(t, err)

		expensesUpdatedCgs, err := StrToContentGroups(account_expenses_update)
		assert.NilError(t, err)

		_, err = accounting.Updateacc(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, expensesAcc.Hash, expensesUpdatedCgs)
		assert.NilError(t, err)

		expensesAccUpdated, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("account"))
		expensesAccUpdatedV, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("accountv"))
	
		expensesAccUpdatedToString, err := accounting.PrintDocument(expensesAccUpdated)
		expensesAccUpdatedVToString, err := accounting.PrintDocument(expensesAccUpdatedV)

		fmt.Println("Account:", expensesAccUpdatedToString)
		fmt.Print("\n\n")
		fmt.Println("Account variable:", expensesAccUpdatedVToString)

	})

	t.Run("Update parent account successfully", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		expensesAcc := trxInfo.Accounts["Expenses"]

		expensesUpdatedCgs, err := StrToContentGroups(account_expenses_update)
		assert.NilError(t, err)

		_, err = accounting.Updateacc(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, expensesAcc.Hash, expensesUpdatedCgs)
		assert.NilError(t, err)

		expensesAccVs, err := docgraph.GetDocumentsWithEdge(env.ctx, &env.api, env.Accounting, expensesAcc, eos.Name("accountv"))
		assert.NilError(t, err)

		expensesAccV := expensesAccVs[0]

		expensesAccUpdatedToString, err := accounting.PrintDocument(expensesAcc)
		expensesAccUpdatedVToString, err := accounting.PrintDocument(expensesAccV)

		fmt.Println("Account:", expensesAccUpdatedToString)
		fmt.Print("\n\n")
		fmt.Println("Account variable:", expensesAccUpdatedVToString)

	})

}

func TestDeleteacc(t *testing.T) {

	t.Run("Test delete leaf account successfully", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)

		ledgerHashStr := CreateTestLedger(env, t)

		pause(t, time.Second, "", "")
	
		ledgerDoc, err := docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, ledgerHashStr)
		assert.NilError(t, err)
	
		fmt.Println("Ledger HASH:", ledgerDoc.Hash)

		expensesAcc, err := CreateAccount(t, env, account_expenses, ledgerDoc.Hash, ledgerDoc.Hash)
		assert.NilError(t, err)

		fmt.Println("Expenses HASH:", expensesAcc.Hash)
		
		_, err = accounting.Deleteacc(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, expensesAcc.Hash)
		assert.NilError(t, err)

		_, err = docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("account"))
		assert.ErrorContains(t, err, "no document with edge found: account")

	})

	t.Run("Test delete leafs in a tree successfully", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		developmentAcc := trxInfo.Accounts["Development"]
		mktingAcc := trxInfo.Accounts["Marketing"]
		expensesAcc := trxInfo.Accounts["Expenses"]

		_, err := docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, developmentAcc.Hash.String())
		assert.NilError(t, err)

		_, err = docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, mktingAcc.Hash.String())
		assert.NilError(t, err)

		expensesAccVs, err := docgraph.GetDocumentsWithEdge(env.ctx, &env.api, env.Accounting, expensesAcc, eos.Name("accountv"))
		assert.NilError(t, err)

		expensesAccV := expensesAccVs[0]
		expensesVToString, err := accounting.PrintDocument(expensesAccV)
		assert.Assert(t, strings.Contains(expensesVToString, "is_leaf : false"))

		fmt.Println("Deleting development account :", developmentAcc.Hash)
		_, err = accounting.Deleteacc(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, developmentAcc.Hash)
		assert.NilError(t, err)

		fmt.Println("Deleting marketing account :", mktingAcc.Hash)
		_, err = accounting.Deleteacc(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, mktingAcc.Hash)
		assert.NilError(t, err)

		_, err = docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, developmentAcc.Hash.String())
		assert.ErrorContains(t, err, "document not found " + developmentAcc.Hash.String())

		_, err = docgraph.LoadDocument(env.ctx, &env.api, env.Accounting, mktingAcc.Hash.String())
		assert.ErrorContains(t, err, "document not found " + mktingAcc.Hash.String())

		
		expensesAccVs, err = docgraph.GetDocumentsWithEdge(env.ctx, &env.api, env.Accounting, expensesAcc, eos.Name("accountv"))
		assert.NilError(t, err)

		expensesAccV = expensesAccVs[0]
		expensesVToString, err = accounting.PrintDocument(expensesAccV)
		assert.Assert(t, strings.Contains(expensesVToString, "is_leaf : true"))

	})

	t.Run("Test delete non leaf accounts (failure expected)", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		expensesAcc := trxInfo.Accounts["Expenses"]

		fmt.Println("Trying to delete expenses account :", expensesAcc.Hash)
		_, err := accounting.Deleteacc(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, expensesAcc.Hash)
		assert.ErrorContains(t, err, "The account ", expensesAcc.Hash, " is not a leaf")

	})

	t.Run("Test delete account with associated components (failure expected)", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		developmentAcc := trxInfo.Accounts["Development"]
		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]
		salaryAcc := trxInfo.Accounts["Salary"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				developmentAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, true)
		assert.NilError(t, err)

		fmt.Println("Trying to delete salary account :", salaryAcc.Hash)
		_, err = accounting.Deleteacc(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, salaryAcc.Hash)
		assert.ErrorContains(t, err, "The account ", salaryAcc.Hash, " already has associated components")

	})

}


func TestAddcurrency(t *testing.T) {

	t.Run("Test add currency successfully", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)

		_, err := accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "1,USD")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,BTC")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,ETH")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "3,HUSD")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "4,WAX")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "5,SEEDS")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "6,TLOS")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "7,EOS")
		assert.NilError(t, err)

		allowedCurrenciesOnChain, err := accounting.GetAllowedCurrencies(env.ctx, &env.api, env.Accounting)
		assert.NilError(t, err)

		assert.Assert(t, CheckAllowedCurrencies(
			[]string{ "USD", "BTC", "ETH", "HUSD", "WAX", "SEEDS", "TLOS", "EOS" },
			allowedCurrenciesOnChain,
		))

	})

}

func TestAddcoinid(t *testing.T) {

	t.Run("An authorized account can add an id to an allowed currency", func(t *testing.T) {

		// Arrange
		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)

		_, err := accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "1,USD")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,BTC")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,ETH")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "3,HUSD")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "5,SEEDS")
		assert.NilError(t, err)

		// Act
		_, err = accounting.AddCoinId(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "5,BTC", "bitcoin")
		_, err = accounting.AddCoinId(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "5,BTC", "ethereum")
		

		// Assert
		assert.NilError(t, err)

		allowedCurrenciesOnChain, err := accounting.GetCoinIds(env.ctx, &env.api, env.Accounting)
		assert.NilError(t, err)

		assert.Assert(t, reflect.DeepEqual(allowedCurrenciesOnChain, []string{ "bitcoin", "ethereum", })  )

	})

}

func TestRemcurrency(t *testing.T) {

	t.Run("Test remove currency successfully", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)

		_, err := accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "1,USD")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,BTC")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "2,ETH")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "3,HUSD")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "4,WAX")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "5,SEEDS")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "6,TLOS")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "7,EOS")
		assert.NilError(t, err)

		_, err = accounting.RemoveCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "3,BTC")
		assert.NilError(t, err)

		_, err = accounting.RemoveCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "3,WAX")
		assert.NilError(t, err)

		_, err = accounting.RemoveCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "3,SEEDS")
		assert.NilError(t, err)

		allowedCurrenciesOnChain, err := accounting.GetAllowedCurrencies(env.ctx, &env.api, env.Accounting)
		assert.NilError(t, err)

		assert.Assert(t, CheckAllowedCurrencies(
			[]string{ "USD", "ETH", "HUSD", "TLOS", "EOS" },
			allowedCurrenciesOnChain,
		))

	})

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
				nil,
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

		developmentAcc := trxInfo.Accounts["Development"]
		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]
		salaryAcc := trxInfo.Accounts["Salary"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				developmentAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
				nil,
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

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Development", []string{}))
		assert.Assert(t, CheckAccountBalances(ledgerToString, "Sales", []string{}))
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

		developmentAcc := trxInfo.Accounts["Development"]
		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]
		salaryAcc := trxInfo.Accounts["Salary"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				developmentAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
				nil,
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
			"[global_USD:-1000.00 USD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Sales", []string{
			"[account_USD:-1000.00 USD]", "[global_USD:-1000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{
			"[account_HUSD:500.00 HUSD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{
			"[global_HUSD:-500.00 HUSD]", "[global_USD:1000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Development", []string{
			"[account_HUSD:-500.00 HUSD]", "[global_HUSD:-500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[account_USD:1000.00 USD]", "[global_USD:1000.00 USD]",
		}))

		pause(t, time.Second * 2, "", "")


		usd3Symbol, _ := eos.StringToSymbol("3,USD")
		btc8Symbol, _ := eos.StringToSymbol("8,BTC")
		tlos4Symbol, _ := eos.StringToSymbol("4,TLOS")

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "8,BTC")
		assert.NilError(t, err)

		_, err = accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "4,TLOS")
		assert.NilError(t, err)

		trxDoc2, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd3Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 80000, Symbol: usd3Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 20000, Symbol: usd3Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: btc8Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: btc8Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 500000, Symbol: tlos4Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 500000, Symbol: tlos4Symbol },
				"DEBIT",
				nil,
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

		edgesLength2["from"] = 8
		edgesLength2["to"] = 8

		assert.Assert(t, CheckTransaction(trxNodeInfo2, trxFields2, edgesLength2))	

		fmt.Println("---------------------------------")

		ledgerToString, err = accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)

		fmt.Print("LEDGER:\n", ledgerToString, "\n\n\n")

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Income", []string{
			"[global_USD:-900.000 USD]", "[global_HUSD:500.00 HUSD]", "[global_BTC:0.00100000 BTC]",
			"[global_TLOS:50.0000 TLOS]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Sales", []string{
			"[account_USD:-980.000 USD]", "[global_USD:-980.000 USD]", "[account_BTC:0.00100000 BTC]", "[global_BTC:0.00100000 BTC]",
			"[account_TLOS:50.0000 TLOS]", "[global_TLOS:50.0000 TLOS]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{
			"[account_HUSD:500.00 HUSD]", "[global_HUSD:500.00 HUSD]","[account_USD:80.000 USD]", "[global_USD:80.000 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{
			"[global_HUSD:-500.00 HUSD]", "[global_USD:900.000 USD]", "[global_BTC:-0.00100000 BTC]",
			"[global_TLOS:-50.0000 TLOS]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Development", []string{
			"[account_HUSD:-500.00 HUSD]", "[global_HUSD:-500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[account_USD:900.000 USD]", "[global_USD:900.000 USD]", "[account_BTC:-0.00100000 BTC]", "[global_BTC:-0.00100000 BTC]",
			"[account_TLOS:-50.0000 TLOS]", "[global_TLOS:-50.0000 TLOS]",
		}))

		fmt.Print("\n\n\n")

	})

	t.Run("Test update without approval", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
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

		developmentAcc := trxInfo.Accounts["Development"]
		salaryAcc := trxInfo.Accounts["Salary"]

		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxUpdatedDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				developmentAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
				nil,
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

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
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

		developmentAcc := trxInfo.Accounts["Development"]
		salaryAcc := trxInfo.Accounts["Salary"]

		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxUpdatedDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				developmentAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
				nil,
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
			"[global_USD:-1000.00 USD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Sales", []string{
			"[account_USD:-1000.00 USD]", "[global_USD:-1000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Salary", []string{
			"[account_HUSD:500.00 HUSD]", "[global_HUSD:500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Expenses", []string{
			"[global_HUSD:-500.00 HUSD]", "[global_USD:1000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Development", []string{
			"[account_HUSD:-500.00 HUSD]", "[global_HUSD:-500.00 HUSD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[account_USD:1000.00 USD]", "[global_USD:1000.00 USD]",
		}))

		pause(t, time.Second * 2, "", "")

	})

	t.Run("Test insert transaction with event, without approval", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]

		eventCgs, err := StrToContentGroups(event_1)
		assert.NilError(t, err)

		_, err = accounting.Event(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, eventCgs)
		assert.NilError(t, err)

		eventDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("event"))
		assert.NilError(t, err)

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				eventDoc.Hash,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)
		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)
		assert.NilError(t, err)

		componentDocs, err := docgraph.GetDocumentsWithEdge(env.ctx, &env.api, env.Accounting, eventDoc, "component")
		assert.NilError(t, err)

		componentDoc := componentDocs[0]

		fmt.Println(componentDoc.Hash.String())
		cmptToString, err := accounting.PrintDocument(componentDoc)
		fmt.Println(cmptToString)

		eventDoc2s, err := docgraph.GetDocumentsWithEdge(env.ctx, &env.api, env.Accounting, componentDoc, "event")
		assert.NilError(t, err)

		eventDoc2 := eventDoc2s[0]

		assert.Assert(t, eventDoc2.Hash.String() == eventDoc.Hash.String())

	})

	t.Run("Test insert transaction with event and approval", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		eventCgs, err := StrToContentGroups(event_1)
		assert.NilError(t, err)

		_, err = accounting.Event(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, eventCgs)
		assert.NilError(t, err)

		eventDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("event"))
		assert.NilError(t, err)

		developmentAcc := trxInfo.Accounts["Development"]
		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]
		salaryAcc := trxInfo.Accounts["Salary"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				eventDoc.Hash,
			},
			accounting.TrxComponent{
				salaryAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				developmentAcc.Hash.String(), 
				eos.Asset{ Amount: 50000, Symbol: husd2Symbol},
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Upserttrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, true)
		assert.NilError(t, err)

		componentDocs, err := docgraph.GetDocumentsWithEdge(env.ctx, &env.api, env.Accounting, eventDoc, "component")
		assert.NilError(t, err)

		componentDoc := componentDocs[0]

		fmt.Println(componentDoc.Hash.String())
		cmptToString, err := accounting.PrintDocument(componentDoc)
		fmt.Println(cmptToString)

		eventDoc2s, err := docgraph.GetDocumentsWithEdge(env.ctx, &env.api, env.Accounting, componentDoc, "event")
		assert.NilError(t, err)

		eventDoc2 := eventDoc2s[0]

		assert.Assert(t, eventDoc2.Hash.String() == eventDoc.Hash.String())

	})

	t.Run("Test failures", func(t *testing.T) {

		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	

		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		usd3Symbol, _ := eos.StringToSymbol("3,USD")

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 10000000, Symbol: usd3Symbol },
				"CREDIT",
				nil,
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
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd3Symbol },
				"CREDIT",
				nil,
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
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 10000, Symbol: usd3Symbol },
				"CREDIT",
				nil,
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

func TestCrryconvtrx(t *testing.T) {

	t.Run("An authorized account can save a transaction for balancing two currencies", func(t *testing.T) {

		// Arrange
		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	
	
		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 500000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: husd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		
		// Act
		_, err = accounting.Crryconvtrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)
		assert.NilError(t, err)


		// Assert
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

	})

	t.Run("An authorized account can approve a transaction for balancing two currencies", func(t *testing.T) {

		// Arrange
		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	
	
		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		btc3Symbol, _ := eos.StringToSymbol("3,BTC")

		_, err := accounting.AddCurrency(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, "3,BTC")
		assert.NilError(t, err)

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 500000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: btc3Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		
		// Act
		_, err = accounting.Crryconvtrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, true)
		assert.NilError(t, err)


		// Assert
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

		ledgerToString, err := accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)
		
		fmt.Println("---------------------------------")
		fmt.Println(ledgerToString)

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[global_USD:5000.00 USD]","[account_USD:5000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Sales", []string{
			"[global_BTC:-100.000 BTC]","[account_BTC:-100.000 BTC]",
		}))

	})

	t.Run("An authorized account can update a non approved transaction", func(t *testing.T) {

		// Arrange
		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	
	
		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 500000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: husd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Crryconvtrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)
		assert.NilError(t, err)

		trxFromChainDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)


		trxDoc, err = createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 200000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: husd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		
		// Act
		_, err = accounting.Crryconvtrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, trxFromChainDoc.Hash, trxDoc.ContentGroups, false)
		assert.NilError(t, err)


		// Assert
		trxFromChainDoc, err = docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
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

	})

	t.Run("An authorized account can update and approve a transaction", func(t *testing.T) {

		// Arrange
		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	
	
		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 500000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: husd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		_, err = accounting.Crryconvtrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)
		assert.NilError(t, err)

		trxFromChainDoc, err := docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
		assert.NilError(t, err)


		trxDoc, err = createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 200000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: husd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		
		// Act
		_, err = accounting.Crryconvtrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, trxFromChainDoc.Hash, trxDoc.ContentGroups, true)
		assert.NilError(t, err)


		// Assert
		trxFromChainDoc, err = docgraph.GetLastDocumentOfEdge(env.ctx, &env.api, env.Accounting, eos.Name("transaction"))
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

		ledgerToString, err := accounting.PrintLedger(env.ctx, &env.api, env.Accounting, ledgerDoc)	
		assert.NilError(t, err)
		
		fmt.Println("---------------------------------")
		fmt.Println(ledgerToString)

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Marketing", []string{
			"[global_USD:2000.00 USD]","[account_USD:2000.00 USD]",
		}))

		assert.Assert(t, CheckAccountBalances(ledgerToString, "Sales", []string{
			"[global_HUSD:-1000.00 HUSD]","[account_HUSD:-1000.00 HUSD]",
		}))

	})

	t.Run("If the transaction has more than 2 currencies, the transaction is invalid", func(t *testing.T) {

		// Arrange
		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	
	
		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]
		husd2Symbol := trxInfo.Currencies["HUSD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 500000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: husd2Symbol },
				"CREDIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: husd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		// Act
		_, err = accounting.Crryconvtrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)

		// Assert
		assert.ErrorContains(t, err, "a currency conversion must have 2 components")

	})

	t.Run("If the 2 currencies are the same, the transaction is invalid", func(t *testing.T) {

		// Arrange
		teardownTestCase := setupTestCase(t)
		defer teardownTestCase(t)	
	
		env := SetupEnvironment(t)
		trxInfo := SetupTrxTestInfo(env, t)

		ledgerDoc := trxInfo.Ledger

		salesAcc := trxInfo.Accounts["Sales"]
		mktingAcc := trxInfo.Accounts["Marketing"]

		usd2Symbol := trxInfo.Currencies["USD2"]

		trxDoc, err := createTrx([]accounting.TrxComponent{
			accounting.TrxComponent{
				mktingAcc.Hash.String(), 
				eos.Asset{ Amount: 500000, Symbol: usd2Symbol },
				"DEBIT",
				nil,
			},
			accounting.TrxComponent{
				salesAcc.Hash.String(), 
				eos.Asset{ Amount: 100000, Symbol: usd2Symbol },
				"CREDIT",
				nil,
			},
		}, &ledgerDoc)

		assert.NilError(t, err)

		// Act
		_, err = accounting.Crryconvtrx(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1, make([]byte, 0), trxDoc.ContentGroups, false)

		// Assert
		assert.ErrorContains(t, err, "a currency conversion must use 2 different currencies, provided only USD")

	})

}
