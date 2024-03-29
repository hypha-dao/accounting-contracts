package accounting

import (
	"context"
	"fmt"
	"strings"
	"strconv"

	"crypto/sha256"
	"encoding/hex"

	"github.com/golang-collections/collections/stack"

	eostest "github.com/digital-scarcity/eos-go-test"
	eos "github.com/eoscanada/eos-go"
	"github.com/hypha-dao/document-graph/docgraph"
)

type createLedger struct {
	Creator    eos.AccountName         `json:"creator"`
	LedgerInfo []docgraph.ContentGroup `json:"ledger_info"`
}

type createAccount struct {
	Creator     eos.AccountName         `json:"creator"`
	AccountInfo []docgraph.ContentGroup `json:"account_info"`
}

type updateAccount struct {
	Updater eos.AccountName `json:"updater"`
	AccountHash eos.Checksum256 `json:"account_hash"`
	AccountInfo []docgraph.ContentGroup `json:"account_info"`
}

type deleteAccount struct {
	Deleter eos.AccountName `json:"updater"`
	AccountHash eos.Checksum256 `json:"account_hash"`
}

type transact struct {
	Issuer          eos.AccountName         `json:"issuer"`
	TransactionInfo []docgraph.ContentGroup `json:"trx_info"`
}

type createTrx struct {
	Creator          eos.AccountName         `json:"creator"`
	TransactionInfo []docgraph.ContentGroup  `json:"trx_info"`
}

type updateTrx struct {
	Updater          eos.AccountName 				 `json:"updater"`
	TransactionHash eos.Checksum256  				 `json:"trx_hash"`
	TransactionInfo []docgraph.ContentGroup  `json:"trx_info"`
}

type balanceTrx struct {
	Issuer          eos.AccountName 				 `json:"issuer"`
	TransactionHash eos.Checksum256  				 `json:"trx_hash"`
}

type setSetting struct {
	Setting string             `json:"setting"`
	Value   docgraph.FlexValue `json:"value"`
}

type remSetting struct {
	Setting string `json:"setting"`
}

type trustAccount struct {
	Account eos.AccountName `json:"account"`
}

type addCurrency struct {
	Issuer eos.AccountName `json:"issuer"`
	Currency eos.Symbol `json:"currency"`
}

type addCoinid struct {
	Issuer eos.AccountName `json:"issuer"`
	Currency eos.Symbol `json:"currency"`
	Id string `json:"id"`
}

type remCurrency struct {
	Authorizer eos.AccountName `json:"authorizer"`
	Currency eos.Symbol `json:"currency"`
}

type cursor struct {
	Key        uint64 `json:"key"`
	Source     string `json:"source"`
	LastCursor string `json:"last_cursor"`
}

type stackNode struct {
	Node	docgraph.Document `json:"document"`
	Level int `json:"level"`
}

type TrxComponent struct {
	AccountHash string `json:"account"`
	Amount eos.Asset `json:"amount"`
	Type string `json:"type"`
	EventHash eos.Checksum256 `json:"event_hash"`
}

type TrxTestInfo struct {
	Ledger docgraph.Document
	Accounts map[string]docgraph.Document
	Currencies map[string]eos.Symbol
}

type upsertTrx struct {
	Issuer eos.AccountName `json:"issuer"`
	TrxHash eos.Checksum256 `json:"trx_hash"`
	TrxInfo []docgraph.ContentGroup `json:"trx_info"`
	Approve bool `json:"approve"`
}

type ComponentNodeInfo struct {
	ComponentNode docgraph.Document
	Edges map[string][]docgraph.Edge
}

type TrxNodeInfo struct {
	TrxNode docgraph.Document
	Edges map[string][]docgraph.Edge
	Components []ComponentNodeInfo
}

type deleteTrx struct {
	Deleter eos.AccountName `json:"deleter"`
	TrxHash eos.Checksum256 `json:"trx_hash"`
}

type ExRateEntry struct {
	From	eos.SymbolCode 	`json:"from"`
	To 		eos.SymbolCode 	`json:"to"`
	Date	eos.TimePoint		`json:"date"`
	Rate 	eos.Int64				`json:"exrate"`
}

type addExchRates struct {
	ExchangeRates []ExRateEntry	`json:"exchange_rates"`
}

type ExRateRow struct {
	Id		eos.Uint64		`json:"id"`
	Date	eos.TimePoint	`json:"date"`
	ToCurrency	string	`json:"to"`
	Rate	eos.Float64		`json:"rate"`
}


func AddLedger(ctx context.Context, api *eos.API, contract, creator eos.AccountName, ledger []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("addledger"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(createLedger{
			Creator:    creator,
			LedgerInfo: ledger,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

// Creates an account
func CreateAcct(ctx context.Context, api *eos.API, contract, creator eos.AccountName, account []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("createacc"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(createAccount{
			Creator:     creator,
			AccountInfo: account,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func Updateacc(ctx context.Context, api *eos.API, contract, updater eos.AccountName, accountHash eos.Checksum256, accountInfo []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("updateacc"),
		Authorization: []eos.PermissionLevel{
			{Actor: updater, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(updateAccount{
			Updater:     updater,
			AccountHash: accountHash,
			AccountInfo: accountInfo,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func Deleteacc(ctx context.Context, api *eos.API, contract, deleter eos.AccountName, accountHash eos.Checksum256) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("deleteacc"),
		Authorization: []eos.PermissionLevel{
			{Actor: deleter, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(deleteAccount{
			Deleter:     deleter,
			AccountHash: accountHash,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func CreateTrxWe(ctx context.Context, api *eos.API, contract, creator eos.AccountName, trx []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("createtrxwe"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(createTrx{
			Creator:          creator,
			TransactionInfo: trx,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func Upserttrx(ctx context.Context, api *eos.API, contract, issuer eos.AccountName, trxHash eos.Checksum256, trxInfo []docgraph.ContentGroup, approve bool) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("upserttrx"),
		Authorization: []eos.PermissionLevel{
			{ Actor: issuer, Permission: eos.PN("active") },
		},
		ActionData: eos.NewActionData(upsertTrx{
			Issuer: issuer,
			TrxHash: trxHash,
			TrxInfo: trxInfo,
			Approve: approve,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)

}

func Crryconvtrx(ctx context.Context, api *eos.API, contract, issuer eos.AccountName, trxHash eos.Checksum256, trxInfo []docgraph.ContentGroup, approve bool) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("crryconvtrx"),
		Authorization: []eos.PermissionLevel{
			{ Actor: issuer, Permission: eos.PN("active") },
		},
		ActionData: eos.NewActionData(upsertTrx{
			Issuer: issuer,
			TrxHash: trxHash,
			TrxInfo: trxInfo,
			Approve: approve,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)

}

func Deletetrx(ctx context.Context, api *eos.API, contract, deleter eos.AccountName, trxHash eos.Checksum256) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("deletetrx"),
		Authorization: []eos.PermissionLevel{
			{ Actor: deleter, Permission: eos.PN("active") },
		},
		ActionData: eos.NewActionData(deleteTrx{
			Deleter: deleter,
			TrxHash: trxHash,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)

}

func CreateTrx(ctx context.Context, api *eos.API, contract, creator eos.AccountName, trx []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("createtrx"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(createTrx{
			Creator:          creator,
			TransactionInfo: trx,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func BalanceTrx(ctx context.Context, api *eos.API, contract, issuer eos.AccountName, trx_hash eos.Checksum256) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("balancetrx"),
		Authorization: []eos.PermissionLevel{
			{Actor: issuer, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(balanceTrx{
			Issuer:						issuer,
			TransactionHash:  trx_hash,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func UpdateTrx(ctx context.Context, api *eos.API, contract, updater eos.AccountName, trx_hash eos.Checksum256,  trx []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("updatetrx"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(updateTrx{
			Updater:          updater,
			TransactionHash:  trx_hash,
			TransactionInfo: trx,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func SetSetting(ctx context.Context, api *eos.API, contract eos.AccountName, setting string, value docgraph.FlexValue) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("setsetting"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(setSetting{
			Setting: setting,
			Value:   value,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func RemSetting(ctx context.Context, api *eos.API, contract eos.AccountName, setting string) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("remsetting"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(remSetting{
			Setting: setting,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func AddTrustedAccount(ctx context.Context, api *eos.API, contract eos.AccountName, account eos.AccountName) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("addtrustacnt"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(trustAccount{
			Account: account,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func RemTrustedAccount(ctx context.Context, api *eos.API, contract eos.AccountName, account eos.AccountName) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("remtrustacnt"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(trustAccount{
			Account: account,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func AddCurrency(ctx context.Context, api *eos.API, contract eos.AccountName, issuer eos.AccountName, currency string) (string, error) {

	symbol, err := eos.StringToSymbol(currency)

	if err != nil {
		return "error", fmt.Errorf("error adding currency: %s", err)
	}

	fmt.Println("Adding currency: ", currency)

	actions := []*eos.Action{{
		Account: contract,
		Name: eos.ActN("addcurrency"),
		Authorization: []eos.PermissionLevel {
			{ Actor: issuer, Permission: eos.PN("active") },
		},
		ActionData: eos.NewActionData(addCurrency {
			Issuer: issuer,
			Currency: symbol,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)

}

func AddCoinId(ctx context.Context, api *eos.API, contract eos.AccountName, issuer eos.AccountName, currency, id string) (string, error) {

	symbol, err := eos.StringToSymbol(currency)

	if err != nil {
		return "error", fmt.Errorf("error removing currency: %s", err)
	}

	fmt.Println("Updating coin id: ", currency)

	actions := []*eos.Action{{
		Account: contract,
		Name: eos.ActN("addcoinid"),
		Authorization: []eos.PermissionLevel {
			{ Actor: issuer, Permission: eos.PN("active") },
		},
		ActionData: eos.NewActionData(addCoinid {
			Issuer: issuer,
			Currency: symbol,
			Id: id,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)

}

func RemoveCurrency(ctx context.Context, api *eos.API, contract eos.AccountName, authorizer eos.AccountName, currency string) (string, error) {

	symbol, err := eos.StringToSymbol(currency)

	if err != nil {
		return "error", fmt.Errorf("error removing currency: %s", err)
	}

	fmt.Println("Removing currency: ", currency)

	actions := []*eos.Action{{
		Account: contract,
		Name: eos.ActN("remcurrency"),
		Authorization: []eos.PermissionLevel {
			{ Actor: authorizer, Permission: eos.PN("active") },
		},
		ActionData: eos.NewActionData(remCurrency {
			Authorizer: authorizer,
			Currency: symbol,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)

}


//Check with permissions
func Event(ctx context.Context, api *eos.API, contract, issuer eos.AccountName, trx []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("newevent"),
		Authorization: []eos.PermissionLevel{
			{Actor: issuer, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(transact{
			Issuer:          issuer,
			TransactionInfo: trx,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}

func AddExchRates(ctx context.Context, api *eos.API, contract eos.AccountName, exchangeRates []ExRateEntry) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("addexchrates"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(addExchRates{
			ExchangeRates:	exchangeRates,
		}),
	}}

	return eostest.ExecTrx(ctx, api, actions)

}

func GetLastCursor(ctx context.Context, api *eos.API, contract eos.AccountName) (string, error) {

	var request eos.GetTableRowsRequest
	request.Code = string(contract)
	request.Scope = string(contract)
	request.Table = "cursors"
	request.Limit = 1
	request.Reverse = true
	request.JSON = true
	response, err := api.GetTableRows(ctx, request)
	if err != nil {
		return "", fmt.Errorf("get table rows: %v", err)
	}

	var cursors []cursor

	err = response.JSONToStructs(&cursors)
	if err != nil {
		return "", fmt.Errorf("json to structs: %v", err)
	}

	if len(cursors) == 0 {
		return "", fmt.Errorf("cursor not found: %v", err)
	}

	return cursors[0].LastCursor, nil
}

func GetCursorFromSource(ctx context.Context, api *eos.API, contract eos.AccountName, source string) (string, error) {

	hashBytes := sha256.Sum256([]byte(source))
	hashStr := hex.EncodeToString(hashBytes[:])

	var request eos.GetTableRowsRequest
	request.Code = string(contract)
	request.Scope = string(contract)
	request.Table = "cursors"
	request.Index = "2"
	request.KeyType = "sha256"
	request.LowerBound = hashStr
	request.UpperBound = hashStr
	request.Limit = 1
	request.Reverse = true
	request.JSON = true
	response, err := api.GetTableRows(ctx, request)
	if err != nil {
		return "", fmt.Errorf("get table rows %v: %v", hashStr, err)
	}

	var cursors []cursor

	err = response.JSONToStructs(&cursors)
	if err != nil {
		return "", fmt.Errorf("json to structs %v: %v", hashStr, err)
	}

	if len(cursors) == 0 {
		return "", fmt.Errorf("cursor not found %v: %v", hashStr, err)
	}

	return cursors[0].LastCursor, nil
}


func PrintLedger (ctx context.Context, api *eos.API, contract eos.AccountName, ledger docgraph.Document) (string, error) {

	balancesToString := ""
	dfs := stack.New()
	accountDocuments, err := docgraph.GetDocumentsWithEdge(ctx, api, contract, ledger, "account")
	
	if err != nil {
		return "", fmt.Errorf("could not retrieve account's children")
	}

	for _, childDocument := range accountDocuments {
		dfs.Push(stackNode{childDocument, 0})
	}

	for dfs.Len() > 0 {
		node := dfs.Pop().(stackNode)
		accountDocument := node.Node

		padding := strings.Repeat("\t", node.Level)

		accountDocVariables, err := docgraph.GetDocumentsWithEdge(ctx, api, contract, accountDocument, "accountv")

		if err != nil {
			return "", fmt.Errorf("could not retrieve account name %v", err)
		}

		accountDocVariable := accountDocVariables[0];
		vDetailsGroup, err := accountDocVariable.GetContentGroup("details")

		if err != nil {
			return "", fmt.Errorf("could not retrieve details %v", err)
		}

		accountName, err := vDetailsGroup.GetContent("account_name")
		isLeaf, err := vDetailsGroup.GetContent("is_leaf")

		if err != nil {
			return "", fmt.Errorf("could not retrieve details from account variable: %v", err)
		}

		balancesDocuments, err := docgraph.GetDocumentsWithEdge(ctx, api, contract, accountDocument, "balances")

		if err != nil {
			return "", fmt.Errorf("could not retrieve balance document %v", err)
		}

		balancesToString += "\n" + padding + "Account:" + accountName.String()
		//fmt.Println(padding, "Account name: ", accountName)

		for _, balanceDocument := range balancesDocuments {
			balancesContentGroup, err := balanceDocument.GetContentGroup("balances")

			if err != nil {
				return "", fmt.Errorf("could not retrieve balance group")
			}

			balancesToString += ", Balances: "

			for _, content := range *balancesContentGroup {
				if content.Label != "content_group_label" {
					balancesToString += "[" + content.Label + ":" + content.Value.String() + "]"
					// fmt.Printf("%s%s : %s\n", padding, content.Label, content.Value.String())
				}
			}

			balancesToString += " endl" + ", isLeaf: " + isLeaf.String()
		}

		accountDocuments, err := docgraph.GetDocumentsWithEdge(ctx, api, contract, accountDocument, "account")

		if err != nil {
			return "", fmt.Errorf("could not retrieve account's children")
		}

		for _, childDocument := range accountDocuments {
			dfs.Push(stackNode{childDocument, node.Level + 1})
		}

	}

	return balancesToString, nil

}

func PrintDocument (document docgraph.Document) (string, error) {

	docToString := ""

	docToString += "ID:" + strconv.FormatUint(document.ID, 10)
	docToString += "\nHash:" + document.Hash.String()
	docToString += "\nCreator:" + string(document.Creator)

	docToString += "\nContent Groups:"

	for i, contentGroup := range document.ContentGroups {
		docToString += "\n" + strconv.Itoa(i) + ":"

		for _, contentItem := range contentGroup {
			docToString += "\n" + contentItem.Label + " : " + contentItem.Value.String()
		}
	}

	return docToString, nil

}

func GetAllEdgesForDocument (ctx context.Context, api *eos.API, contract eos.AccountName, document docgraph.Document) (map[string][]docgraph.Edge, error) {
	
	edges := make(map[string][]docgraph.Edge)

	fromEdges, err := docgraph.GetEdgesFromDocument(ctx, api, contract, document)

	if err != nil {
		return nil, fmt.Errorf("could not retrieve from edges: %v", err)
	}

	fmt.Println("edges length:", len(fromEdges))

	edges["from"] = fromEdges

	toEdges, err := docgraph.GetEdgesToDocument(ctx, api, contract, document)

	if err != nil {
		return nil, fmt.Errorf("could not retrieve to edges: %v", err)
	}

	edges["to"] = toEdges

	return edges, nil

}

func GetTrxNodeInfo (ctx context.Context, api *eos.API, contract eos.AccountName, transaction docgraph.Document) (TrxNodeInfo, error) {

	GetAllEdgesForDocument(ctx, api, contract, transaction)

	trxEdges, err := GetAllEdgesForDocument(ctx, api, contract, transaction)

	if err != nil {
		return TrxNodeInfo{}, err
	}

	fromEdges := trxEdges["from"]
	var components []ComponentNodeInfo

	for _, edge := range fromEdges {

		if edge.EdgeName == "component" {
			comptDoc, err := docgraph.LoadDocument(ctx, api, contract, edge.ToNode.String())
			comptEdges, err := GetAllEdgesForDocument(ctx, api, contract, comptDoc)

			if err != nil {
				return TrxNodeInfo{}, err
			}

			components = append(components, ComponentNodeInfo {
				ComponentNode: comptDoc,
				Edges: comptEdges,
			})
		}

	}

	return TrxNodeInfo {
		TrxNode: transaction,
		Edges: trxEdges,
		Components: components,
	}, nil

}

func GetAllowedCurrencies (ctx context.Context, api *eos.API, contract eos.AccountName) ([]eos.Symbol, error) {

	settingsDoc, err := docgraph.GetLastDocumentOfEdge(ctx, api, contract, "settings")

	if err != nil {
		return []eos.Symbol{}, err
	}

	allowedCurrenciesGroup, err := settingsDoc.GetContentGroup("allowed_currencies")

	if err != nil {
		return []eos.Symbol{}, err
	}

	var allowedCurrencies []eos.Symbol

	for _, currency := range *allowedCurrenciesGroup {

		if currency.Label == "allowed_currency" {
			currencyAsset, err := currency.Value.Asset()

			if err != nil {
				return []eos.Symbol{}, err
			}

			allowedCurrencies = append(allowedCurrencies, currencyAsset.Symbol)
		}
	}

	return allowedCurrencies, nil
}

func GetCoinIds (ctx context.Context, api *eos.API, contract eos.AccountName) ([]string, error) {

	settingsDoc, err := docgraph.GetLastDocumentOfEdge(ctx, api, contract, "settings")

	if err != nil {
		return []string{}, err
	}

	allowedCurrenciesGroup, err := settingsDoc.GetContentGroup("allowed_currencies")

	if err != nil {
		return []string{}, err
	}

	var allowedCurrencies []string

	for _, c := range *allowedCurrenciesGroup {

		if strings.Contains(c.Label, "_ID") {
			coinId := c.Value.String()

			if err != nil {
				return []string{}, err
			}

			allowedCurrencies = append(allowedCurrencies, coinId)
		}
	}

	return allowedCurrencies, nil

}


func GetExchangeRates(ctx context.Context, api *eos.API, contract eos.AccountName, from, to string) ([]ExRateRow, error) {

	toSymbolCode, _ := eos.StringToSymbolCode(to)

	delimiter := eos.Uint128{
		Lo: uint64(0),
		Hi: uint64(toSymbolCode),
	}

	var request eos.GetTableRowsRequest
	request.Code = string(contract)
	request.Scope = from
	request.Table = "exrates"
	request.LowerBound = delimiter.String()
	request.Limit = 10000
	request.Index = "2"
	request.KeyType = "i128"
	request.JSON = true
	response, err := api.GetTableRows(ctx, request)
	
	if err != nil {
		return []ExRateRow{}, fmt.Errorf("fail to get table rows %v", err)
	}

	var rows []ExRateRow

	err = response.JSONToStructs(&rows)
	if err != nil {
		return []ExRateRow{}, fmt.Errorf("json to structs %v", err)
	}

	return rows, nil

}

