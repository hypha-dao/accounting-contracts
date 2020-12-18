#pragma once

namespace hypha {

constexpr auto ROOT_NODE = "root_node";
constexpr auto OPENING_BALANCES = "opening_balances";
constexpr auto ACCOUNT_NAME = "account_name";
constexpr auto LEDGER_ACCOUNT = "ledger_account";
constexpr auto ACCOUNT_TYPE = "account_type";
constexpr auto PARENT_ACCOUNT = "parent_account";
constexpr auto OPENING_BALANCE_PREFIX = "opening_balance";
constexpr auto TRX_MEMO = "trx_memo";
constexpr auto TRX_DATE = "trx_date";
constexpr auto TRX_LEDGER = "trx_ledger";
constexpr auto COMPONENT_AMMOUNT = "amount";
constexpr auto COMPONENT_MEMO = "memo";
constexpr auto COMPONENT_ACCOUNT = "account";

inline size_t
createID() {
  static size_t id = 0;
  return id++;
}

template<class T>
size_t
getClassID(T = T()) 
{
  static size_t id = createID();
  return id;
}

}