#pragma once

#include <eosio/eosio.hpp>

namespace hypha {

using eosio::name;

constexpr auto ROOT_NODE = "root_node";
constexpr auto DETAILS = "details";
constexpr auto SYSTEM = "system";
constexpr auto NAME_LABEL = "node_label";
constexpr auto TYPE_LABEL = "type";
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
constexpr auto SETTINGS = "settings";
constexpr auto UPDATE_DATE = "update_date";
constexpr auto SETTINGS_DATA = "settings_data";
constexpr auto TRUSTED_ACCOUNTS_GROUP = "trusted_accounts";
constexpr auto TRUSTED_ACCOUNT_LABEL = "trusted_account";
constexpr auto SETTINGS_EDGE = name("settings");

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