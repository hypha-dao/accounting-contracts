#include <accounting.hpp>

#include <document_graph/util.hpp>

#include "constants.hpp"
#include "table_wrapper.hpp"

namespace hypha {

ACTION
accounting::addledger(name creator, ContentGroups& ledger_info)
{
  require_auth(get_self());

  Document ledger(get_self(), creator, std::move(ledger_info));

  Document openingsAcc(get_self(), creator, getOpeningsAccount(ledger.getHash()));
  
  Edge(get_self(), creator, ledger.getHash(), openingsAcc.getHash(), name("account"));
  Edge(get_self(), creator, openingsAcc.getHash(), ledger.getHash(), name("ownedby"));

  Edge(get_self(), creator, getRoot().getHash(), ledger.getHash(), name("ledger"));
}

ACTION
accounting::create(name creator, ContentGroups& account_info)
{ 
  require_auth(get_self());

  ContentWrapper contentWrap(account_info);
  
  auto parentHash = contentWrap.getOrFail("details", PARENT_ACCOUNT)->getAs<checksum256>();

  auto accountName = contentWrap.getOrFail("details", ACCOUNT_NAME)->getAs<string>();
  
  auto accountEdges = m_documentGraph.getEdgesFrom(parentHash, name("account"));

  TableWrapper<document_table> docs(get_self(), get_self().value);

  check(docs.contains_by<"idhash"_n>(parentHash), 
        "The parent document doesn't exists: " + readableHash(parentHash));
  //Check if there isn't already an account with the same name
  //Alternative option is to have a document with only the account name
  //or to add some field like `name` to the document structure
  for (const auto& edge : accountEdges) {
    
    auto account = docs.get_by<name("idhash")>(edge.to_node);

    ContentWrapper cg(account.getContentGroups());

    auto name = cg.getOrFail("details", ACCOUNT_NAME)->getAs<string>();

    check(name != accountName, "There is already an account with name: " + name);
  }

  Document account(get_self(), creator, std::move(account_info));
  
  Edge(get_self(), creator, parentHash, account.getHash(), name("account"));
  Edge(get_self(), creator, account.getHash(), parentHash, name("ownedby"));
  
  if (auto [idx, balances] = contentWrap.getGroup(OPENING_BALANCES);
      balances)
  {
    for (const auto& [label, value] : *balances) 
    {
      if (label != CONTENT_GROUP_LABEL)
      {

      }
    }
  }
}

/**
* Transaction structure
* group: details
* transaction_id : int64_t
* transaction_date : time_point
* memo: string
* [*] exchange_rate_id : int64_t 
* 
* 
* group: component_x
*  
**/
ACTION
accounting::transact(name issuer, ContentGroups& transaction_info)
{
  require_auth(issuer);


}

ContentGroups
accounting::getOpeningsAccount(checksum256 parent)
{
  ContentGroup details {
    Content(CONTENT_GROUP_LABEL, "details"),
    Content(ACCOUNT_NAME, "Opening Balances"),
    Content(ACCOUNT_TYPE, 1),
    Content(PARENT_ACCOUNT, parent)
  };

  ContentGroup openings {
    Content(CONTENT_GROUP_LABEL, "opening_balances"),
    Content("opening_balance_usd", asset(0, symbol("USD", 4))),
    Content("opening_balance_usd", asset(0, symbol("BTC", 12)))
  };

  return {details, openings};
}

const Document& 
accounting::getRoot()
{
  //This assumes the root document won't change
  static Document rootDoc = Document::getOrNew(get_self(), 
                                               get_self(),
                                               Content(ROOT_NODE, get_self()));
  
  return rootDoc;
}

}