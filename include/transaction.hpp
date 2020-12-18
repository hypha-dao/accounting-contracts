#pragma once

#include <string>
#include <vector>

#include <eosio/eosio.hpp>

#include <document_graph/content_wrapper.hpp>

#include "constants.hpp"

namespace hypha {

using eosio::checksum256;
using eosio::name;
using eosio::asset;
using eosio::time_point;
using std::string;
using std::vector;

class Transaction
{
 public:

  /**
  * @brief Builds a transaction object from a given ContentGroups
  */
  Transaction(ContentGroups& trxInfo);

  class Component
  {
   public:
    checksum256 account;
    asset amount;
    string memo;
  };

  void
  verifyBalanced();

  inline const vector<Component>&
  getComponents() const 
  {
    return m_components;
  }

  inline ContentGroup
  getDetails() const
  {
    return {
      Content{CONTENT_GROUP_LABEL, "details"},
      Content{TRX_MEMO, m_memo},
      Content{TRX_DATE, m_date},
      Content{TRX_LEDGER, m_ledger}
    };
  }
  
 private:
  string m_memo;
  time_point m_date;
  checksum256 m_ledger;
  vector<Component> m_components;
  //name m_signature;
};

}