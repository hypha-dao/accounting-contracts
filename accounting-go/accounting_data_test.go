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

const transaction_test_1 = `
{
	"content_groups": 
	[
			[
					{
							"label": "content_group_label",
							"value": [
									"string",
									"header"
							]
					},
					{
							"label": "trx_date",
							"value": [
									"time_point",
									"2020-12-17T21:45:11.500"
							]
					},
					{
							"label": "trx_ledger",
							"value": [
									"string",
									"abc"
							]
					},
					{
							"label": "trx_memo",
							"value": [
									"string",
									"Test transaction 1"
							]
					}
		 ],
		 [
					{
						"label": "content_group_label",
						"value": [
								"string",
								"component"
						]
					},
					{
						"label": "memo",
						"value": [
							"string",
							"Test component"
						]
					},
					{
						"label": "account_a",
						"value": [
							"string",
							"abc"
						]
					},
					{
							"label": "amount",
							"value": [
									"asset",
									"1000.00 USD"
							]
					}
		 ],
		 [
					{
						"label": "content_group_label",
						"value": [
								"string",
								"component"
						]
					},
					{
						"label": "memo",
						"value": [
							"string",
							"Test component"
						]
					},
					{
						"label": "account_b",
						"value": [
							"string",
							"abc"
						]
					},
					{
							"label": "amount",
							"value": [
									"asset",
									"-1000.00 USD"
							]
					}
		 ]
	]
}
`
const account_openings_tester = `
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
									"Income"
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
									"2000.00 USD"
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

const account_mkting = `
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
									"Marketing"
							]
					},
					{
							"label": "account_type",
							"value": [
									"int64",
									1
							]
					}
		 ]
	]
}`

const account_income = `
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
									"Income"
							]
					},
					{
							"label": "account_type",
							"value": [
									"int64",
									1
							]
					}
		 ]
	]
}
`

const account_expenses = `
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
									"Expenses"
							]
					},
					{
							"label": "account_type",
							"value": [
									"int64",
									1
							]
					}
		 ]
	]
}
`

const account_salary = `
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
									"Salary"
							]
					},
					{
							"label": "account_type",
							"value": [
									"int64",
									1
							]
					}
		 ]
	]
}`
