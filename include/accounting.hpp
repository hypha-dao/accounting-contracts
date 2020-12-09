#include <eosio/eosio.hpp>

#include <document_graph/document_graph.hpp>
#include <document_graph/content_group.hpp>

namespace hypha {

using namespace eosio;

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
  transact(name issuer, ContentGroups& transaction_info);

  /**
  * Gets the root document of the graph
  */
  const Document& 
  getRoot();


  ContentGroups
  getOpeningsAccount(checksum256 parent);
 private:

  DocumentGraph m_documentGraph{get_self()};
};

}
