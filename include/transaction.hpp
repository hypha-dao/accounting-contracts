#include <string>
#include <vector>

#include <eosio/eosio.hpp>

#include <document_graph/content_wrapper.hpp>

namespace hypha {

using eosio::checksum256;
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
  Transaction(ContentGroups trxInfo);

  class Component
  {
   public:
    checksum256 account;
    asset amount;
    string memo;
  };

  bool isBalanced();
  
  string memo;
  time_point date;
  checksum256 ledger;
  vector<Component> components;
};

}