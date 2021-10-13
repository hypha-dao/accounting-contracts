package accounting_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/alexeyco/simpletable"
	eostest "github.com/digital-scarcity/eos-go-test"
	"github.com/eoscanada/eos-go"
	"github.com/hypha-dao/accounting-go"
	"github.com/hypha-dao/dao-go"
	"github.com/hypha-dao/document-graph/docgraph"
	"gotest.tools/assert"
)

//const defaultKey = "5KQwrPbwdL6PhXujxW37FSSQZ1JiwsST4cqQzDeyXtP79zkvFD3"

var accountingHome, accountingWasm, accountingAbi, devHome, daoHome, daoWasm, daoAbi, tokenHome, tokenWasm, tokenAbi, tdHome, tdWasm, tdAbi string
var treasuryHome, treasuryWasm, treasuryAbi, monitorHome, monitorWasm, monitorAbi string
var seedsHome, escrowWasm, escrowAbi, exchangeWasm, exchangeAbi string

const testingEndpoint = "http://localhost:8888"

type Member struct {
	Member eos.AccountName
	Doc    docgraph.Document
}

type Environment struct {
	ctx context.Context
	api eos.API

	DAO         eos.AccountName
	Accounting  eos.AccountName
	HusdToken   eos.AccountName
	HyphaToken  eos.AccountName
	HvoiceToken eos.AccountName
	AuthorizedAccount1 eos.AccountName

	Whale Member
	Root  docgraph.Document

	VotingDurationSeconds int64
	HyphaDeferralFactor   int64
	SeedsDeferralFactor   int64

	NumPeriods     int
	PeriodDuration time.Duration

	Members []Member
}

func envHeader() *simpletable.Header {
	return &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "Variable"},
			{Align: simpletable.AlignCenter, Text: "Value"},
		},
	}
}

func (e *Environment) String() string {
	table := simpletable.New()
	table.Header = envHeader()

	kvs := make(map[string]string)
	kvs["DAO"] = string(e.DAO)
	kvs["HUSD Token"] = string(e.HusdToken)
	kvs["HVOICE Token"] = string(e.HvoiceToken)
	kvs["HYPHA Token"] = string(e.HyphaToken)
	kvs["Whale"] = string(e.Whale.Member)
	kvs["Voting Duration (s)"] = strconv.Itoa(int(e.VotingDurationSeconds))
	kvs["HYPHA deferral X"] = strconv.Itoa(int(e.HyphaDeferralFactor))
	kvs["SEEDS deferral X"] = strconv.Itoa(int(e.SeedsDeferralFactor))

	for key, value := range kvs {
		r := []*simpletable.Cell{
			{Align: simpletable.AlignLeft, Text: key},
			{Align: simpletable.AlignRight, Text: value},
		}
		table.Body.Cells = append(table.Body.Cells, r)
	}

	return table.String()
}

func SetupEnvironment(t *testing.T) *Environment {

	home, exists := os.LookupEnv("HOME")
	if exists {
		devHome = home
	} else {
		devHome = "."
	}
	devHome = devHome + ""
	// devHome = "src/"
	devHome = "../.."

	accountingHome = devHome + "/accounting-contracts"
	accountingWasm = accountingHome + "/build/accounting/accounting.wasm"
	accountingAbi = accountingHome + "/build/accounting/accounting.abi"

	var env Environment

	env.api = *eos.New(testingEndpoint)
	// api.Debug = true
	env.ctx = context.Background()

	keyBag := &eos.KeyBag{}
	err := keyBag.ImportPrivateKey(env.ctx, eostest.DefaultKey())
	assert.NilError(t, err)

	env.api.SetSigner(keyBag)

	env.VotingDurationSeconds = 2
	env.SeedsDeferralFactor = 100
	env.HyphaDeferralFactor = 25

	env.PeriodDuration, _ = time.ParseDuration("6s")
	env.NumPeriods = 10

	env.Accounting, _ = eostest.CreateAccountFromString(env.ctx, &env.api, "accounting", eostest.DefaultKey())
	env.AuthorizedAccount1, _ = eostest.CreateAccountFromString(env.ctx, &env.api, "authacct1111", eostest.DefaultKey())

	env.DAO, _ = eostest.CreateAccountFromString(env.ctx, &env.api, "dao.hypha", eostest.DefaultKey())

	_, env.HusdToken, _ = eostest.CreateAccountWithRandomKey(env.ctx, &env.api, "husd.hypha")
	_, env.HvoiceToken, _ = eostest.CreateAccountWithRandomKey(env.ctx, &env.api, "hvoice.hypha")
	_, env.HyphaToken, _ = eostest.CreateAccountWithRandomKey(env.ctx, &env.api, "token.hypha")

	// t.Log("Deploying DAO contract to 		: ", env.DAO)
	// setCodeActions, err := system.NewSetCode(env.DAO, daoWasm)
	// _, err = eostest.ExecTrx(env.ctx, &env.api, []*eos.Action{setCodeActions})
	// assert.NilError(t, err)

	// setAbiActions, err := system.NewSetABI(env.DAO, daoAbi)
	// _, err = eostest.ExecTrx(env.ctx, &env.api, []*eos.Action{setAbiActions})
	// assert.NilError(t, err)

	t.Log("Deploying Accounting contract to 		: ", env.Accounting)
	_, err = eostest.SetContract(env.ctx, &env.api, env.Accounting, accountingWasm, accountingAbi)
	assert.NilError(t, err)
	// _, err = eostest.SetContract(env.ctx, &env.api, env.DAO, daoWasm, daoAbi)
	// assert.NilError(t, err)

	env.Root, err = CreateRoot(env.ctx, &env.api, env.Accounting, env.Accounting)
	assert.NilError(t, err)

	_, err = accounting.AddTrustedAccount(env.ctx, &env.api, env.Accounting, env.Accounting)
	assert.NilError(t, err)

	_, err = accounting.AddTrustedAccount(env.ctx, &env.api, env.Accounting, env.AuthorizedAccount1)
	assert.NilError(t, err)
	
	return &env
}

func SetupMember(t *testing.T, ctx context.Context, api *eos.API,
	contract, telosDecide eos.AccountName, memberName string, hvoice eos.Asset) (Member, error) {

	t.Log("Creating and enrolling new member  		: ", memberName, " 	with voting power	: ", hvoice.String())
	memberAccount, err := eostest.CreateAccountFromString(ctx, api, memberName, eostest.DefaultKey())
	assert.NilError(t, err)

	_, err = dao.RegVoter(ctx, api, telosDecide, memberAccount)
	assert.NilError(t, err)

	_, err = dao.Mint(ctx, api, telosDecide, contract, memberAccount, hvoice)
	assert.NilError(t, err)

	_, err = dao.Apply(ctx, api, contract, memberAccount, "apply to DAO")
	assert.NilError(t, err)

	_, err = dao.Enroll(ctx, api, contract, contract, memberAccount)
	assert.NilError(t, err)

	memberDoc, err := docgraph.GetLastDocument(ctx, api, contract)
	assert.NilError(t, err)

	memberNameFV, err := memberDoc.GetContent("member")
	assert.NilError(t, err)
	assert.Equal(t, eos.AN(string(memberNameFV.Impl.(eos.Name))), memberAccount)

	return Member{
		Member: memberAccount,
		Doc:    memberDoc,
	}, nil
}
