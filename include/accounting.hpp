#pragma once

#include <string_view>
#include <vector>

#include <eosio/eosio.hpp>
#include <eosio/crypto.hpp>

#include <document_graph/document_graph.hpp>
#include <document_graph/content_wrapper.hpp>

#include "eosio_utils.hpp"

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

  TABLE exchange_rate
  {
    uint64_t id;
    time_point date;
    symbol_code from_currency;
    symbol_code to_currency;
    checksum256 trx_origin;
    float rate;
    bool invalidated;

    EOSLIB_SERIALIZE(exchange_rate, (id)(date)(from_currency)(to_currency)(trx_origin)(rate)(invalidated))

    uint64_t primary_key() const { return id; }
    checksum256 by_trx_origin() const { return trx_origin; }
  };

  using exchange_rates_table = multi_index<"exrates"_n, exchange_rate,
                                           indexed_by<name("trxorigin"), const_mem_fun<exchange_rate, checksum256, &exchange_rate::by_trx_origin>>>;

  TABLE currency
  {
    symbol_code code;
    uint64_t primary_key() const { return code.raw(); }
  };

  using currencies_table = multi_index<"currencies"_n, currency>;

  /**
  * Creates the root document (useful for testing)
  */ 
  ACTION
  createroot();

  /**
  * Adds a ledger account to the graph
  *
  * 
  */
  ACTION
  addledger(name creator, ContentGroups& ledger_info);

  /**
  * Adds an account to the graph
  */
  ACTION 
  create(name creator, ContentGroups& account_info);

  /**
   * Stores the components and transaction information in the graph
   */
  ACTION
  transact(name issuer, ContentGroups& trx_info);

  ACTION
  newunrvwdtrx(name issuer, ContentGroups trx_info);

  /**
  * Adds a setting in the settings document or replaces it if the setting already
  * exits
  */
  ACTION
  setsetting(string setting, Content::FlexValue value);

  /**
  * Deletes a setting from the settings document
  */ 
  ACTION
  remsetting(string setting);
  
  /**
  * Adds an account to the trusted accounts group. Necesary to trigger unrvwd trx action
  */  
  ACTION
  addtrustacnt(name account);

  /**
  * Remove an account from the trusted accounts group.
  */
  ACTION
  remtrustacnt(name account);

  /**
  * Clears the unrvwdtrxs from the graph
  */
  ACTION
  clearunrvwd(int64_t max_removable_trx);

  /**
  * Clears the data
  */
  ACTION
  clean(ContentGroups& tables);

  /**
  * Gets the root document of the graph
  */
  static const Document& 
  getRoot();

  static name
  getName();

  static ContentGroup
  getSystemGroup(const char* nodeName, const char* type);

  ContentGroups
  getOpeningsAccount(checksum256 parent);

  ContentGroups
  getEquityAccount(checksum256 parent);

  bool
  isCurrencySupported(symbol currency);

  const std::vector<symbol_code>&
  getSupportedCurrencies();

  void
  requireTrusted(name account);
  
  /**
  * Retreives the hash of the Unreviewed Transactions Bucket document
  */
  checksum256
  getUnreviewedTrxBucket();
 private:

  ContentGroup
  getTrxHeader(string memo, time_point date, checksum256 ledger);

  checksum256
  getOpeningsHash(checksum256 parent);

  ContentGroup
  getTrxComponent(checksum256 account, 
                  string memo, 
                  asset amount, 
                  string label = "component");

  /**
  * @brief Creates a parent->child relationship with edges between accounts 
  */
  void 
  parent(name creator, 
         checksum256 parent, 
         checksum256 child, 
         string_view fromToEdge = "account", 
         string_view toFromEdge = "ownedby");

  DocumentGraph m_documentGraph{get_self()};

  static name g_contractName;
};

}
