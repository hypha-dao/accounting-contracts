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

  ACTION
  fixcmp(std::vector<checksum256> documents)
  {
    require_auth(get_self());
    const size_t maxDocs = 10;

    for (size_t i = 0; i < std::min(maxDocs, documents.size()); ++i) {
      Document docs(get_self(), documents[i]);
      auto cw = docs.getContentWrapper();
      
      auto type = cw.getOrFail(SYSTEM, TYPE_LABEL)->getAs<string>();

      EOS_CHECK(
        type == "component",
        util::to_str("Wrong document type: ", type)
      )

      bool hasNewContents = false;

      auto details = cw.getGroupOrFail(DETAILS);

      if (!cw.exists(DETAILS, COMPONENT_FROM)) {
        cw.insertOrReplace(*details, Content{COMPONENT_FROM, ""});
        hasNewContents = true;
      }

      if (!cw.exists(DETAILS, COMPONENT_TO)) {
        cw.insertOrReplace(*details, Content{COMPONENT_TO, ""});
        hasNewContents = true;
      }

      if (!cw.exists(DETAILS, COMPONENT_TAG_TYPE)) {
        cw.insertOrReplace(*details, Content{COMPONENT_TAG_TYPE, "DEBIT"});
        hasNewContents = true;
      }

      if (hasNewContents) {
        m_documentGraph.updateDocument(
          get_self(),
          docs.getHash(),
          cw.getContentGroups()
        );
      }
    }

    if (documents.size() > maxDocs) {
      std::vector<checksum256> nextBatch(documents.begin() + maxDocs, documents.end());
      eosio::action(
        eosio::permission_level{get_self(), "active"_n},
        get_self(),
        "fixcmp"_n,
        std::make_tuple(nextBatch)
      ).send();
    }
  }

  /**
  * Creates the root document (useful for testing)
  */ 
  ACTION
  createroot(std::string notes);

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
  createacc(name creator, ContentGroups& account_info);

  /**
   * @brief Creates an unapproved transaction
   * 
   * @param creator User that created
   * @param trx_info 
   */
  ACTION
  createtrx(name creator, ContentGroups& trx_info);

  /**
   * @brief Creates a transaction with an empty component linked to an event
   * 
   * @param creator 
   * @param trx_info 
   */
  ACTION
  createtrxwe(name creator, ContentGroups& trx_info);

  /**
   * @brief Updates an unapproved transaction by adding, deleting or modifying components
   * 
   * @param trx_info 
   * @return ACTION 
   */
  ACTION
  updatetrx(name updater, checksum256 trx_hash, ContentGroups& trx_info);

  /**
   * Stores the components and transaction information in the graph
   */
  ACTION
  balancetrx(name issuer, checksum256 trx_hash);

  ACTION
  newevent(name issuer, ContentGroups trx_info);

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
   * @brief Updates account information
   * 
   * @param account_hash
   * @param account_info  
   * @return ACTION 
   */
  ACTION
  updateacc(name updater, checksum256 account_hash, ContentGroups account_info);
  
  /**
  * Adds an account to the trusted accounts group. Necesary to trigger newevent action
  */  
  ACTION
  addtrustacnt(name account);

  /**
  * Remove an account from the trusted accounts group.
  */
  ACTION
  remtrustacnt(name account);

  ACTION
  deletetrx(name deleter, checksum256 trx_hash);

  /**
   * @brief Binds an event with a component document
   * 
   * @param event_hash 
   * @param component_hash 
   * @return ACTION 
   */
  ACTION
  bindevent(name updater, checksum256 event_hash, checksum256 component_hash);

  /**
   * @brief Unbinds an event from a component document
   * 
   * @param event_hash 
   * @param component_hash 
   * @return ACTION 
   */
  ACTION
  unbindevent(name updater, checksum256 event_hash, checksum256 component_hash);
    
  /**
  * Clears the events from the graph
  */
  ACTION
  clearevent(int64_t max_removable_trx);

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
  getSystemGroup(std::string nodeName, std::string type);

  bool
  isCurrencySupported(symbol currency);

  const std::vector<symbol_code>&
  getSupportedCurrencies();

  void
  requireTrusted(name account);
  
  /**
  * Retreives the hash of the Events Bucket document
  */
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

  std::string
  getAccountPath(std::string account, checksum256 parent, const checksum256& ledger);

  //Adds the given amount to an specific account balances
  void 
  addAssetToAccount(checksum256 account, asset amount);

  void
  recalculateGlobalBalances(checksum256 account, checksum256 ledger);

  /**
   * @brief Adds a list of balances to a given account balances
   * 
   * @param account 
   * @param balances 
   * @param accCW 
   */
  void
  addToBalance(Document& balancesDoc, 
               const std::vector<Balance>& balances);

  Document
  getAccountBalances(checksum256 account);

  /**
   * @brief Get's the global balances of an account
   * 
   * @param account 
   * @return std::vector<asset> [Balances]
   */
  std::map<std::string, asset>
  getAccountGlobalBalances(checksum256 account);

  /**
   * @brief Get's the local balances of an account
   * 
   * @param account 
   * @return std::vector<asset> [Balances]
   */
  std::map<std::string, asset>
  getAccountLocalBalances(checksum256 account);

  void 
  setGlobalBalances(Document& balancesDoc,
                    std::map<std::string, asset>& balances);

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

  /**
  * @brief Creates a parent --> child & parent <-- child
  * edges relationship between accounts 
  */
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
