package accounting

import (
	"context"

	eostest "github.com/digital-scarcity/eos-go-test"
	"github.com/eoscanada/eos-go"
	"github.com/hypha-dao/document/docgraph"
)

type createLedger struct {
	Creator    eos.AccountName         `json:"creator"`
	LedgerInfo []docgraph.ContentGroup `json:"ledger_info"`
}

type createAccount struct {
	Creator     eos.AccountName         `json:"creator"`
	AccountInfo []docgraph.ContentGroup `json:"account_info"`
}

type transact struct {
	Issuer          eos.AccountName         `json: "issuer"`
	TransactionInfo []docgraph.ContentGroup `json: "trx_info"`
}

type setSetting struct {
	Setting string             `json: "setting"`
	Value   docgraph.FlexValue `json: "value"`
}

type remSetting struct {
	Setting string `json: "setting"`
}

type trustAccount struct {
	Account eos.AccountName `json: "account`
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
		Name:    eos.ActN("create"),
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

func Transact(ctx context.Context, api *eos.API, contract, issuer eos.AccountName, trx []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("transact"),
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(transact{
			Issuer:          issuer,
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

//Check with permissions
func UnreviewedTrx(ctx context.Context, api *eos.API, contract, issuer eos.AccountName, trx []docgraph.ContentGroup) (string, error) {

	actions := []*eos.Action{{
		Account: contract,
		Name:    eos.ActN("newunrvwdtrx"),
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
