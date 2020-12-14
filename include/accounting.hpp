#include <eosio/eosio.hpp>

#include <document_graph/document_graph.hpp>
#include <document_graph/content_wrapper.hpp>
#include <vector>

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
  using contract::contract;

  DECLARE_DOCUMENT_GRAPH(accounting)

  ACTION
  addledger(name creator, ContentGroups& ledger_info);

  /**
  * Adds an account to the graph
  */
  ACTION 
  create(name creator, ContentGroups& account_info);

  ACTION
  transact(name issuer, ContentGroups& trx_info);

  /**
  * Gets the root document of the graph
  */
  const Document& 
  getRoot() const;


  ContentGroups
  getOpeningsAccount(checksum256 parent);

  ContentGroups
  getEquityAccount(checksum256 parent);

  static bool
  isCurrencySupported(symbol currency);

  static const std::vector<symbol_code>&
  getSupportedCurrencies();
 private:

  ContentGroup
  getTrxHeader(string memo, time_point date, checksum256 ledger);

  checksum256
  getOpeningsHash(checksum256 parent);

  ContentGroup
  getTrxComponent(checksum256 account, string memo, asset amount);

  /**
  * @brief Creates a parent->child relationship with edges between accounts 
  */
  void 
  parent(name creator, checksum256 parent, checksum256 child);

  DocumentGraph m_documentGraph{get_self()};
};

}
