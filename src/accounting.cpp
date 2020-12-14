#include <accounting.hpp>

#include <algorithm>

#include <document_graph/util.hpp>

#include "constants.hpp"
#include "container_utils.hpp"
#include "table_wrapper.hpp"
#include "transaction.hpp"

namespace hypha {

ACTION
accounting::addledger(name creator, ContentGroups& ledger_info)
{
  require_auth(get_self());

  Document ledger(get_self(), creator, std::move(ledger_info));

  //Create default Equity
  Document equityAcc(get_self(), creator, getEquityAccount(ledger.getHash()));

  //Create default Equity::OpeningsAccount
  Document openingsAcc(get_self(), creator, getOpeningsAccount(equityAcc.getHash()));
  
  parent(creator, ledger.getHash(), equityAcc.getHash());

  parent(creator, equityAcc.getHash(), openingsAcc.getHash());

  Edge(get_self(), creator, getRoot().getHash(), ledger.getHash(), name("ledger"));
}

ACTION
accounting::create(name creator, ContentGroups& account_info)
{ 
  require_auth(get_self());

  ContentWrapper contentWrap(account_info);
  
  auto parentHash = contentWrap.getOrFail("details", PARENT_ACCOUNT)->getAs<checksum256>();

  auto accountName = contentWrap.getOrFail("details", ACCOUNT_NAME)->getAs<string>();

  auto ledger = contentWrap.getOrFail("details", LEDGER_ACCOUNT)->getAs<checksum256>();
  
  auto accountEdges = m_documentGraph.getEdgesFrom(parentHash, name("account"));

  TableWrapper<document_table> docs(get_self(), get_self().value);

  check(docs.contains_by<"idhash"_n>(parentHash), 
        "The parent document doesn't exists: " + readableHash(parentHash));
  
  //Check if there isn't already an account with the same name
  for (const auto& edge : accountEdges) {
    
    auto account = docs.get_by<"idhash"_n>(edge.to_node);

    ContentWrapper cg(account.getContentGroups());

    auto name = cg.getOrFail("details", ACCOUNT_NAME)->getAs<string>();

    check(util::toLowerCase(std::move(name)) != util::toLowerCase(accountName), 
          "There is already an account with name: " + accountName);
  }

  //Create the account
  Document account(get_self(), creator, std::move(account_info));
  
  parent(creator, parentHash, account.getHash());
  
  if (auto [idx, balances] = contentWrap.getGroup(OPENING_BALANCES);
      balances) {

    ContentGroups transaction{getTrxHeader("Opening Balances", 
                                           current_time_point(),
                                           ledger)};

    auto openingBalances = getOpeningsHash(ledger);

    for (const auto& content : *balances) {
      
      auto& [label, value] = content;

      if (util::containsPrefix(label, OPENING_BALANCE_PREFIX)) {
        
        asset componentAmount = content.getAs<asset>();
       
        check(componentAmount.is_valid(), 
              "Invalid asset: " + componentAmount.to_string());

        check(isCurrencySupported(componentAmount.symbol), 
              string("Unsupported currency: ") + componentAmount.symbol.code().to_string());

        ContentGroup component {
          Content{CONTENT_GROUP_LABEL, "Component"},
          Content{COMPONENT_ACCOUNT, account.getHash()},
          Content{COMPONENT_MEMO, "Opening Balance"},
          Content{COMPONENT_AMMOUNT, componentAmount}
        };

        componentAmount.set_amount(-componentAmount.amount);

        ContentGroup componentOB {
          Content{CONTENT_GROUP_LABEL, "Component"},
          Content{COMPONENT_ACCOUNT, openingBalances},
          Content{COMPONENT_MEMO, "Opening Balance"},
          Content{COMPONENT_AMMOUNT, componentAmount}
        };

        transaction.emplace_back(std::move(component));
      }
      else if (label != CONTENT_GROUP_LABEL) {
        check(false, "Wrong format for opening_balances account [" + label + "]");
      }
    }

    transact(creator, transaction);
  }
}

/**
* Transaction structure
* group: header
* transaction_date : time_point
* memo: string
* ledger: ledger checksum
* 
* [components]
* group: component
* account: account hash
* memo: string
* amount: asset
*  
**/
ACTION
accounting::transact(name issuer, ContentGroups& trx_info)
{
  require_auth(issuer);

  //ContentWrapper trx(trx_info);

  Transaction trx(trx_info);

  check(trx.isBalanced(), "Sum of all components in transactions doesn't add to 0");


}

ContentGroups
accounting::getOpeningsAccount(checksum256 parent)
{
  ContentGroup details {
    Content(CONTENT_GROUP_LABEL, "details"),
    Content(ACCOUNT_NAME, "Opening Balances"),
    Content(ACCOUNT_TYPE, ACCOUNT_GROUP::kEquity),
    Content(PARENT_ACCOUNT, parent)
  };

  /*
  ContentGroup openings {
    Content(CONTENT_GROUP_LABEL, "opening_balances"),
    Content("opening_balance_usd", asset(0, symbol("USD", 4))),
    Content("opening_balance_btc", asset(0, symbol("BTC", 12)))
  };
  */

  return {details/*, openings*/};
}

ContentGroups
accounting::getEquityAccount(checksum256 parent)
{
  ContentGroup details {
    Content(CONTENT_GROUP_LABEL, "details"),
    Content(ACCOUNT_NAME, "Equity"),
    Content(ACCOUNT_TYPE, ACCOUNT_GROUP::kEquity),
    Content(PARENT_ACCOUNT, parent)
  };

  /*
  ContentGroup openings {
    Content(CONTENT_GROUP_LABEL, "opening_balances"),
    Content("opening_balance_usd", asset(0, symbol("USD", 4))),
    Content("opening_balance_usd", asset(0, symbol("BTC", 12)))
  }; */

  return {details/*, openings*/};
}

checksum256 
accounting::getOpeningsHash(checksum256 ledger)
{
  auto equityHash = Document::hashContents(getEquityAccount(ledger));
  auto openingBalancesHash = Document::hashContents(getOpeningsAccount(equityHash));

  return openingBalancesHash;
}

ContentGroup
accounting::getTrxHeader(string memo, time_point date, checksum256 ledger)
{
  return {
    Content{CONTENT_GROUP_LABEL, "header"},
    Content{TRX_MEMO, std::move(memo)},
    Content{TRX_DATE, date},
    Content{TRX_LEDGER, ledger}
  };
}

void 
accounting::parent(name creator, checksum256 parent, checksum256 child)
{
  Edge(get_self(), creator, parent, child, name("account"));
  Edge(get_self(), creator, child, parent, name("ownedby"));
}

const Document& 
accounting::getRoot() const
{
  //This assumes the root document won't change
  static Document rootDoc = Document::getOrNew(get_self(), 
                                               get_self(),
                                               Content(ROOT_NODE, get_self()));
  
  return rootDoc;
}

bool
accounting::isCurrencySupported(symbol sym)
{
  //TODO: Add supported assets (?)
  /*
  auto symCode = sym.code();
  auto& currencies = getSupportedCurrencies();
  return std::find(currencies.begin(), currencies.end(), symCode) != currencies.end();
  */

  return true;
}

//Might want to use symbol if discriminating by precision as well
const std::vector<symbol_code>&
accounting::getSupportedCurrencies()
{
  static std::vector<symbol_code> supportedCurrencies
  {
    symbol_code{"USD"},
    symbol_code{"USDT"},
    symbol_code{"BTC"},
    symbol_code{"ETC"},
    symbol_code{"HUSD"},
    symbol_code{"TUSD"}
  };

  return supportedCurrencies;
}

}