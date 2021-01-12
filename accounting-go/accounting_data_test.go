package accounting_test

const ledger_tester = `
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
					"label": "owner",
					"value": [
							"name",
							"tester"
					]
				},
				{
					"label": "name",
					"value": [
							"string",
							"common"
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

const transaction_test_implied = `
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
									"Test Implied Transaction"
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
							"New marketing campain"
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
							"Restaurant"
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
									"2.4 BTC"
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
							"Week salary"
						]
					},
					{
						"label": "account_c",
						"value": [
							"string",
							"abc"
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

const account_food = `
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
									"Food"
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

const unreviewd_trx_1 = `
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
						"label": "transaction_id",
						"value": [
							"string",
							"0"
						]
					},
					{
						"label": "from",
						"value": [
							"string",
							"s120222ef34012fjk39fk290"
						]
					},
					{
						"label": "treasury_id",
						"value": [
							"string",
							"btc-treasury-1"
						]
					},
					{
						"label": "to",
						"value": [
							"string",
							"s12x12tref34012fjk39fk2a0"
						]
					},
					{
						"label": "quantity",
						"value": [
							"string",
							"0.25"
						]
					},
					{
						"label": "currency",
						"value": [
							"string",
							"BTC"
						]
					},
					{
						"label": "timestamp",
						"value": [
							"string",
							"2021-04-12 21:10:22"
						]
					},
					{
						"label": "usd_value",
						"value": [
							"string",
							"9000.00"
						]
					},
					{
						"label": "memo",
						"value": [
							"string",
							"Monthly fee"
						]
					},
					{
						"label": "chain_id",
						"value": [
							"string",
							"ABCD"
						]
					}
		 ]
	]
}
`
