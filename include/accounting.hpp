#pragma once

#include <string_view>
#include <vector>

#include <eosio/eosio.hpp>

#include <document_graph/document_graph.hpp>
#include <document_graph/content_wrapper.hpp>

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

  /**
  * Creates the root document (useful for testing)
  */ 
  ACTION
  createroot();

  /**
  * Adds a ledger account to the graph
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
  newunrvwdtrx(name creator, ContentGroups trx_info);

  ACTION
  setsetting(string setting, Content::FlexValue value);

  ACTION
  remsetting(string setting);
  
  ACTION
  addtrustacnt(name account);

  ACTION
  remtrustacnt(name account);
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
