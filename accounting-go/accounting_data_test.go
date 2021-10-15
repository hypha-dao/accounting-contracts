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

const unbalanced_trx = `
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
					},
					{
							"label": "trx_name",
							"value": [
								"string",
								"transaction name"
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
					},
					{
						"label": "from",
						"value": [
								"string",
								"test_from"
						]
					},
					{
						"label": "to",
						"value": [
								"string",
								"test_to"
						]
					},
					{
						"label": "type",
						"value": [
								"string",
								"DEBIT"
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
									"1000.00 HUSD"
							]
					},
					{
						"label": "from",
						"value": [
								"string",
								"test_from2"
						]
					},
					{
						"label": "to",
						"value": [
								"string",
								"test_to2"
						]
					},
					{
						"label": "type",
						"value": [
								"string",
								"CREDIT"
						]
					}
		 ]
	]
}
`

const generic_trx = `
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
				"label": "trx_date",
				"value": [
						"time_point",
						"2020-12-17T21:45:11.500"
				]
			},
			{
				"label": "trx_ledger",
				"value": [
						"checksum256",
						"trx_ledger_value"
				]
			},
			{
				"label": "trx_memo",
				"value": [
						"string",
						"Test transaction"
				]
			},
			{
				"label": "trx_name",
				"value": [
					"string",
					"transaction name"
				]
			}
		]
		generic_trx_components
	]
}
`

const generic_trx_component = `
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
		"label": "account",
		"value": [
			"checksum256",
			"component_account"
		]
	},
	{
		"label": "amount",
		"value": [
				"asset",
				"component_amount"
		]
	},
	{
		"label": "from",
		"value": [
				"string",
				"test_from"
		]
	},
	{
		"label": "to",
		"value": [
				"string",
				"test_to"
		]
	},
	{
		"label": "type",
		"value": [
				"string",
				"component_type"
		]
	}
]
`


const transaction_test_we = `
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
							"label": "trx_date",
							"value": [
									"time_point",
									"2021-11-17T21:45:11.500"
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
									"Test transaction We"
							]
					},
					{
						"label": "event",
						"value": [
								"string",
								"abc"
						]
					}
				]
		 ]
}
`

const transaction_test_we_update = `
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
							"label": "trx_date",
							"value": [
									"time_point",
									"2021-11-17T21:45:11.500"
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
									"Test transaction We"
							]
					},
					{
							"label": "id",
							"value": [
									"int64",
									0
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
					},
					{
							"label": "event",
							"value": [
								"string",
								"abc"
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
									"details"
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
					},
					{
						"label": "account_tag_type",
						"value": [
								"string",
								"DEBIT"
						]
					},
					{
						"label": "account_code",
						"value": [
								"string",
								"000111"
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
					},
					{
						"label": "account_tag_type",
						"value": [
								"string",
								"CREDIT"
						]
					},
					{
						"label": "account_code",
						"value": [
								"string",
								"000113"
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
					},
					{
						"label": "account_tag_type",
						"value": [
								"string",
								"DEBIT"
						]
					},
					{
						"label": "account_code",
						"value": [
								"string",
								"000112"
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
					},
					{
						"label": "account_tag_type",
						"value": [
								"string",
								"CREDIT"
						]
					},
					{
						"label": "account_code",
						"value": [
								"string",
								"000114"
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
					},
					{
						"label": "account_tag_type",
						"value": [
								"string",
								"DEBIT"
						]
					},
					{
						"label": "account_code",
						"value": [
								"string",
								"000115"
						]
					}
		 ]
	]
}`

const event_1 = `
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
						"label": "chainId",
						"value": [
							"string",
							"bip122:000000000019d6689c085ae165831e93"
						]
					},
					{
						"label": "source",
						"value": [
							"string",
							"btc-treasury-1"
						]
					},
					{
						"label": "cursor",
						"value": [
							"string",
							"18a835a0d11c91ab6abdd75bf7df1e67deada952b448193e1d4ad76c6e585dfd;0"
						]
					}
		 ]
	]
}
`

const event_2 = `
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
							"2"
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
							"0.5"
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
							"18000.00"
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
						"label": "chainId",
						"value": [
							"string",
							"bip122:000000000019d6689c085ae165831e93"
						]
					},
					{
						"label": "source",
						"value": [
							"string",
							"btc-treasury-2"
						]
					},
					{
						"label": "cursor",
						"value": [
							"string",
							"87a835a0d11c91ab6abdd75bf7df1e67deada952b448193e1d4ad76c6e585bbb;9"
						]
					}
		 ]
	]
}
`

/*
//Trx data
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
					"label": "parent_account",
						"value": [
								"checksum256",
								"4c807227a2c9d7ebe5b22050f6d3f0d4318fcb57904e19e18746ae0309024481"
						]
				},
				{
					"label": "ledger_account",
						"value": [
								"checksum256",
								"4c807227a2c9d7ebe5b22050f6d3f0d4318fcb57904e19e18746ae0309024481"
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
*/
