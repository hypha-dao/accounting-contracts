#pragma once

#include <string_view>
#include <vector>
#include <map>

#include <eosio/eosio.hpp>
#include <eosio/crypto.hpp>

#include <document_graph/document_graph.hpp>
#include <document_graph/content_wrapper.hpp>

#include <constants.hpp>
#include "eosio_utils.hpp"
#include "math_utils.hpp"

namespace hypha {

using namespace eosio;

namespace ACCOUNT_GROUP {
enum E
{
  kAsset,
  kLiability,
  kEquity,
  kRevenue,
  kExpense,
  kGain,
  kLoss
};
}

struct exchange_rate_entry {
  eosio::symbol_code from;
  eosio::symbol_code to;
  eosio::time_point date;
  int64_t exrate;
};

CONTRACT accounting : public contract {
 public:
  //using contract::contract;

  accounting( name self, name first_receiver, datastream<const char*> ds );

  DECLARE_DOCUMENT_GRAPH(accounting)

  TABLE cursor
  {
    uint64_t key;
    string source;
    string last_cursor;

    uint64_t primary_key() const { return key; }

    checksum256 by_source() const { return util::hashString(source); }

    EOSLIB_SERIALIZE(cursor, (key)(source)(last_cursor))
  };

  using cursor_table = multi_index<"cursors"_n, cursor,
                                   indexed_by<"bysource"_n, const_mem_fun<cursor, checksum256, &cursor::by_source>>
                                   >;


  TABLE exchange_rate { // scoped by symbol_code
    uint64_t id;
    time_point date;
    symbol_code to;
    double rate;

    uint64_t primary_key () const { return id; }
    uint128_t by_to_date () const { return (uint128_t(to.raw()) << 64) + uint128_t(date.time_since_epoch().count()); }
  };

  using exchange_rates_table = multi_index<"exrates"_n, exchange_rate,
                                          indexed_by<name("bytodate"), const_mem_fun<exchange_rate, uint128_t, &exchange_rate::by_to_date>>>;


  ACTION
  createroot(std::string notes);

  ACTION
  addledger(name creator, ContentGroups& ledger_info);

  ACTION 
  createacc(const name & creator, ContentGroups& account_info);

  ACTION
  updateacc(const name & updater, const checksum256 & account_hash, ContentGroups & account_info);

  ACTION
  deleteacc(const name & deleter, const checksum256 & account_hash);

  ACTION
  upserttrx(const name & issuer, const checksum256 & trx_hash, ContentGroups & trx_info, bool approve);

  ACTION
  deletetrx(const name & deleter, const checksum256 & trx_hash);

  ACTION
  setsetting(string setting, Content::FlexValue value);

  ACTION
  remsetting(string setting);
  
  ACTION
  addtrustacnt(name account);

  ACTION
  remtrustacnt(name account);

  ACTION
  addcurrency(const name & updater, symbol & currency_symbol);

  ACTION
  remcurrency(const name & authorizer, const symbol & currency_symbol);

  ACTION
  newevent(name issuer, ContentGroups trx_info);

  ACTION
  bindevent(name updater, checksum256 event_hash, checksum256 component_hash);

  ACTION
  unbindevent(name updater, checksum256 event_hash, checksum256 component_hash);

  ACTION
  clearevent(int64_t max_removable_trx);

  ACTION
  addexchrates(std::vector<exchange_rate_entry> exchange_rates);

  ACTION
  clean(ContentGroups& tables);

  ACTION
  reset(int64_t batch_size);


  static const Document& 
  getRoot();

  static name
  getName();

  static ContentGroup
  getSystemGroup(std::string nodeName, std::string type);

  bool
  isAllowedCurrency(const symbol & currency_symbol, const std::vector<uint64_t> & allowed_currencies);

  const std::vector<uint64_t>&
  getAllowedCurrencies();

  void
  requireTrusted(name account);

  void
  createTransaction(const name & issuer, int64_t trxId, ContentGroups & trx_info, bool approve);

  void
  deleteTransaction(const checksum256 & trx_hash);

  bool
  isApproved(const checksum256 & trx_hash);
  
  checksum256
  getEventBucket();


 private:

  struct Balance
  {
    std::string label;
    asset amount;
  };

  void
  createAccountBalance();

  void
  createComponents(checksum256 trx_hash, class Transaction& trx, name creator);

  Document
  getAccountBalances(checksum256 account);

  ContentGroup
  getTrxComponent(checksum256 account, 
                  string memo, 
                  asset amount, 
                  string from, 
                  string to, 
                  string type,
                  string label);

  ContentGroup
  getBalancesSystemGroup(int64_t id);

  Document
  getAccountVariable(const checksum256 & account_hash);

  bool
  hasAssociatedComponents(const checksum256 & account_hash);

  void
  changeAcctBalanceRecursively(
    const checksum256 & account, 
    const checksum256 & ledger, 
    const asset & quantity,
    const bool onlyGlobal
  );

  void 
  parent(name creator, 
         checksum256 parent, 
         checksum256 child, 
         string_view fromToEdge = ACCOUNT_EDGE, 
         string_view toFromEdge = OWNED_BY);

  DocumentGraph m_documentGraph{get_self()};

  static name g_contractName;
};

}
