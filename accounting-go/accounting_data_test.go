package accounting_test

const ledger_tester = `
{
  "content_groups": 
	[
			[
				{
					"label": "owner",
					"value": [
							"name",
							"tester"
					]
				}
			]
	]
}`


const account_tester = `
{
	"content_groups": 
	[
			[
					{
							"label": "content_group_label",
							"value": [
									"string",
									"details"
							]
					},
					{
							"label": "account_name",
							"value": [
									"string",
									"Marketing Expenses"
							]
					},
					{
							"label": "account_type",
							"value": [
									"int64",
									1
							]
					}
		 ],
		 [
					{
						 "label": "content_group_label",
						 "value": [
								 "string",
								 "opening_balances"
								]
					},
					{
							"label": "opening_balance_usd",
							"value": [
									"asset",
									"1.00 USD"
							]
					},
					{
							 "label": "opening_balance_btc",
							 "value": [
									 "asset",
									 "0.50000000 BTC"
								]
					}
		 ]
	]
}`