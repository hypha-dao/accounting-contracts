package accounting

import (
	"context"
	"fmt"

	eostest "github.com/digital-scarcity/eos-go-test"
	"github.com/eoscanada/eos-go"
)


// SetConfigAtt sets a single attribute on the configuration
func SayHi(ctx context.Context, api *eos.API, contract eos.AccountName) (string, error) {

	action := eos.ActN("hi")
	actionData := make(map[string]interface{})
	actionData["nm"] = "test"
	
	actionBinary, err := api.ABIJSONToBin(ctx, contract, eos.Name(action), actionData)
	if err != nil {
		return "error", fmt.Errorf("cannot pack action data")
	}

	actions := []*eos.Action{{
		Account: contract,
		Name:    action,
		Authorization: []eos.PermissionLevel{
			{Actor: contract, Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionDataFromHexData([]byte(actionBinary)),
	}}

	return eostest.ExecTrx(ctx, api, actions)
}