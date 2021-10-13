#pragma once

#include <eosio/eosio.hpp>

namespace hypha {

using eosio::name;

constexpr auto ROOT_NODE = "root_node";
constexpr auto DETAILS = "details";
constexpr auto BALANCES = "balances";
constexpr auto SYSTEM = "system";
constexpr auto NAME_LABEL = "node_label";
constexpr auto TYPE_LABEL = "type";
constexpr auto OPENING_BALANCES = "opening_balances";
constexpr auto ACCOUNT_NAME = "account_name";
constexpr auto ACCOUNT_PATH = "path";
constexpr auto LEDGER_ACCOUNT = "ledger_account";
constexpr auto ACCOUNT_TYPE = "account_type";
constexpr auto ACCOUNT_TAG_TYPE = "account_tag_type";
constexpr auto ACCOUNT_CODE = "account_code";
constexpr auto ACCOUNT_EDGE = "account";
constexpr auto DEBIT_TAG_TYPE = "DEBIT";
constexpr auto CREDIT_TAG_TYPE = "CREDIT";
constexpr auto PARENT_ACCOUNT = "parent_account";
constexpr auto OPENING_BALANCE_PREFIX = "opening_balance";
constexpr auto TRX_LABEL = "Transaction";
constexpr auto TRX_TYPE = "transaction";
constexpr auto TRX_ID = "id";
constexpr auto UNAPPROVED_TRX = "unapproved";
constexpr auto APPROVED_TRX = "approved";
constexpr auto TRX_MEMO = "trx_memo";
constexpr auto TRX_NAME = "trx_name";
constexpr auto TRX_NOTES = "trx_notes";
constexpr auto TRX_DATE = "trx_date";
constexpr auto TRX_LEDGER = "trx_ledger";
constexpr auto TRX_APPROVER = "approved_by";
constexpr auto OWNED_BY = "ownedby";
constexpr auto COMPONENT_AMMOUNT = "amount";
constexpr auto COMPONENT_MEMO = "memo";
constexpr auto COMPONENT_LABEL = "Component";
constexpr auto COMPONENT_TYPE = "component";
constexpr auto COMPONENT_ACCOUNT = "account";
constexpr auto COMPONENT_FROM = "from";
constexpr auto COMPONENT_TO = "to";
constexpr auto COMPONENT_TAG_TYPE = "type";
constexpr auto BALANCE_ACCOUNT = COMPONENT_ACCOUNT;
constexpr auto COMPONENT_DATE = "create_date";
constexpr auto SETTINGS = "settings";
constexpr auto UPDATE_DATE = "update_date";
constexpr auto CREATE_DATE = "create_date";
constexpr auto SETTINGS_DATA = "settings_data";
constexpr auto TRUSTED_ACCOUNTS_GROUP = "trusted_accounts";
constexpr auto TRUSTED_ACCOUNT_LABEL = "trusted_account";
constexpr auto ALLOWED_CURRENCIES_GROUP = "allowed_currencies";
constexpr auto ALLOWED_CURRENCIES_LABEL = "allowed_currency";
constexpr auto SETTINGS_EDGE = name("settings");
constexpr auto TRX_BUCKET_LABEL = "Transactions Bucket";
constexpr auto TRX_BUCKET_EDGE = "trxbucket";
constexpr auto EVENT_BUCKET_EDGE = "eventbucket";
constexpr auto EVENT_BUCKET_LABEL = "Events Bucket";
constexpr auto EVENT_EDGE = "event";
constexpr auto EVENT_LABEL = "Event";
constexpr auto EVENT_SOURCE = "source";
constexpr auto EVENT_CURSOR = "cursor";
constexpr auto BALANCE_UPDATE = "update_date";

constexpr auto MAX_REMOVABLE_DOCS = int64_t(100);

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