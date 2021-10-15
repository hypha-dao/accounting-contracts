#pragma once

#include <string>
#include <vector>

#include <eosio/eosio.hpp>

#include <document_graph/content_wrapper.hpp>
#include <document_graph/document.hpp>

#include "constants.hpp"

namespace hypha {

using eosio::checksum256;
using eosio::name;
using eosio::asset;
using eosio::time_point;
using std::string;
using std::vector;

class DocumentGraph;

class Transaction
{
 public:

  Transaction(ContentGroups& trxInfo);

  Transaction(Document& trxDoc, DocumentGraph& docgraph);

  class Component
  {
   public:
    Component() {}
    Component(ContentGroups& data);
    checksum256 account;
    asset amount;
    string memo;
    string from;
    string to;
    string type;
    std::optional<checksum256> event;
    std::optional<checksum256> hash;
  };

  void
  checkBalanced();

  inline const vector<Component>&
  getComponents() const 
  {
    return m_components;
  }

  inline ContentGroup *
  getDetails() const
  {
    return m_details;
  }

  inline checksum256
  getLedger() const 
  {
    return m_ledger;
  }

  inline int64_t
  getID() const
  {
    return m_id;
  }

  bool
  shouldUpdate(Transaction& original);
    
 private:
  string m_memo;
  string m_name;
  time_point m_date;
  checksum256 m_ledger;
  int64_t m_id;
  vector<Component> m_components;
  ContentGroup * m_details;
  //name m_signature;
};

}