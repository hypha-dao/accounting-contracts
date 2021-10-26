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
  clean(ContentGroups& tables);


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

  std::string
  getAccountPath(std::string account, checksum256 parent, const checksum256& ledger);

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
