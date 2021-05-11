#include "accounting.hpp"

#include <eosio/system.hpp>

#include <algorithm>

#include <document_graph/util.hpp>
#include <logger/logger.hpp>

#include "constants.hpp"
#include "container_utils.hpp"
#include "settings.hpp"
#include "table_wrapper.hpp"
#include "transaction.hpp"

namespace hypha {

name accounting::g_contractName = {};

accounting::accounting(name self, name first_receiver, datastream<const char*> ds)
  : contract(self, first_receiver, ds)
{
  g_contractName = get_self();
}

ACTION 
accounting::createroot()
{
  TRACE_FUNCTION()

  require_auth(get_self());
  getRoot();
}

ACTION
accounting::addledger(name creator, ContentGroups& ledger_info)
{
  TRACE_FUNCTION()

  require_auth(get_self());

  ContentWrapper cw(ledger_info);

  auto& ledgerName = cw.getOrFail(DETAILS, "name")->getAs<string>();
  
  ledger_info.push_back(getSystemGroup(ledgerName.c_str(), "ledger"));

  auto withOpeningsAccount = cw.getOrFail(DETAILS, "create_openings_account")->getAs<int64_t>();

  Document ledger(get_self(), creator, std::move(ledger_info));

  if (withOpeningsAccount == 1) {
    //Create default Equity
    Document equityAcc(get_self(), creator, getEquityAccount(ledger.getHash()));

    // //Create default Equity::OpeningsAccount
    Document openingsAcc(get_self(), creator, getOpeningsAccount(equityAcc.getHash()));
    
    parent(creator, ledger.getHash(), equityAcc.getHash());

    parent(creator, equityAcc.getHash(), openingsAcc.getHash());
  }

  Edge(get_self(), creator, getRoot().getHash(), ledger.getHash(), name("ledger"));
}

ACTION
accounting::create(name creator, ContentGroups& account_info)
{ 
  TRACE_FUNCTION()

  require_auth(get_self());

  ContentWrapper contentWrap(account_info);

  auto [dIdx, details] = contentWrap.getGroup(DETAILS);

  EOS_CHECK(details, "Details group was expected but not found in account info");
  
  auto parentHash = contentWrap.getOrFail(dIdx, PARENT_ACCOUNT).second->getAs<checksum256>();

  auto accountType = contentWrap.getOrFail(dIdx, ACCOUNT_TYPE).second->getAs<int64_t>();

  auto accountName = contentWrap.getOrFail(dIdx, ACCOUNT_NAME).second->getAs<string>();

  auto ledger = contentWrap.getOrFail(dIdx, LEDGER_ACCOUNT).second->getAs<checksum256>();
  
  auto accountEdges = m_documentGraph.getEdgesFrom(parentHash, name("account"));

  TableWrapper<document_table> docs(get_self(), get_self().value);

  EOS_CHECK(docs.contains_by<"idhash"_n>(parentHash), 
        "The parent document doesn't exists: " + readableHash(parentHash));
  
  //Check if there isn't already an account with the same name
  for (const auto& edge : accountEdges) {
    
    auto account = docs.get_by<"idhash"_n>(edge.to_node);

    ContentWrapper cg(account.getContentGroups());

    auto name = cg.getOrFail(DETAILS, ACCOUNT_NAME)->getAs<string>();

    EOS_CHECK(util::toLowerCase(std::move(name)) != util::toLowerCase(accountName), 
          "There is already an account with name: " + accountName);
  }

  //Create the account
  Document account(get_self(), creator, { 
    ContentGroup{
      Content{CONTENT_GROUP_LABEL, DETAILS},
      Content{ACCOUNT_NAME, accountName},
      Content{ACCOUNT_TYPE, accountType},
      Content{PARENT_ACCOUNT, parentHash}
    },
    getSystemGroup(accountName.c_str(), "account"),
  });
  
  parent(creator, parentHash, account.getHash());
  
  //Process opening balances if any
  if (auto [idx, balances] = contentWrap.getGroup(OPENING_BALANCES);
      balances) {

    auto openingBalances = getOpeningsHash(ledger);

    for (const auto& content : *balances) {
      
      auto& [label, value] = content;

      if (util::containsPrefix(label, OPENING_BALANCE_PREFIX)) {
               
        asset componentAmount = content.getAs<asset>();
       
        EOS_CHECK(componentAmount.is_valid(), 
              "Invalid asset: " + componentAmount.to_string());

        EOS_CHECK(isCurrencySupported(componentAmount.symbol), 
              string("Unsupported currency: ") + componentAmount.symbol.code().to_string());

        auto memo = util::to_str("Opening Balance: ", componentAmount.symbol.code());

        ContentGroups transaction{getTrxHeader(memo, 
                                  current_time_point(),
                                  ledger)};

        transaction.emplace_back(getTrxComponent(account.getHash(), 
                                                 memo,
                                                 componentAmount));

        componentAmount.set_amount(-componentAmount.amount);

        transaction.emplace_back(getTrxComponent(openingBalances, 
                                                 memo,
                                                 componentAmount));

        transact(get_self(), transaction);
      }
      else if (label != CONTENT_GROUP_LABEL) {
        EOS_CHECK(false, "Wrong format for opening_balances account [" + label + "]");
      }
    }
  }
}

/**
* Transaction structure
* label: header
* transaction_date : time_point
* memo: string
* 
* [components]
* label: component
* account: account hash checksum256
* memo: string
* amount: asset
*  
**/
ACTION
accounting::transact(name issuer, ContentGroups& trx_info)
{
  TRACE_FUNCTION()

  require_auth(issuer);

  Transaction trx(trx_info);

  std::vector<asset> assets = trx.verifyBalanced();

  TableWrapper<currencies_table> currencies(get_self(), get_self().value);

  for (auto& a : assets) {
    if (!currencies.contains(a.symbol.code().raw())) {
      currencies.insert(get_self(), [a](currency& c){
        c.code = a.symbol.code();
      });
    }
  }

  asset fromCurrency;
  asset toCurrency;

  if (assets.size() == 2) {
    fromCurrency = assets[0];
    toCurrency = assets[1];
  }
  else if (assets.size() == 1) {
    fromCurrency = toCurrency = assets[0];
  }
  else {
    EOS_CHECK(
      false, 
      "Assets size has to be 2 or 1, actual:" + std::to_string(assets.size())
    );
  }

  Document trxDoc(get_self(), issuer, { trx.getDetails(), 
                                        getSystemGroup("transaction", "trx") });

  TableWrapper<exchange_rates_table> exrates(get_self(), get_self().value);

  EOS_CHECK(
    !exrates.contains_by<"trxorigin"_n>(trxDoc.getHash()),
    util::to_str("There is another exchange rate with the same transaction origin: ",
                 readableHash(trxDoc.getHash()))
  );

  EOS_CHECK(
    (fromCurrency.amount * toCurrency.amount) != 0 || 
    (fromCurrency.amount == 0 && toCurrency.amount == 0) ||
    //This is an special case for implied assets. Since the assets is invalid 
    //its code is 0
    (fromCurrency.symbol.raw() == 0),
    util::to_str("Zero sum error. Both assets must be either 0 or different than 0. [", 
                 fromCurrency, toCurrency, "]")
  );

  exrates.insert(get_self(), [&](exchange_rate& exrate){
    exrate.id = exrates.get_next_pk();
    exrate.date = current_time_point();
    exrate.from_currency = fromCurrency.symbol.code();
    exrate.to_currency = toCurrency.symbol.code();
    exrate.trx_origin = trxDoc.getHash();
    exrate.invalidated = false;
    exrate.rate = fromCurrency.amount == 0 ? 1.f : util::calculateRate(fromCurrency, toCurrency);
  });

  for (auto& compnt : trx.getComponents()) {
    Document compntAcct(get_self(), compnt.account);

    Document compntDoc(get_self(), issuer, { getTrxComponent(compnt.account, 
                                                             compnt.memo, 
                                                             compnt.amount,
                                                             DETAILS),
                                             getSystemGroup("component", "component") });

    parent(issuer, trxDoc.getHash(), compntDoc.getHash(), "component", "transaction");

    Edge compntToAcc(get_self(), issuer, compntDoc.getHash(), compntAcct.getHash(), "account"_n);
    
    Edge::getOrNew(get_self(), issuer, compntAcct.getHash(), trxDoc.getHash(), "transaction"_n);
  }
}

ACTION
accounting::newunrvwdtrx(name issuer, ContentGroups trx_info) 
{
  TRACE_FUNCTION()
  
  require_auth(issuer);
  
  //Check if account is trusted
  requireTrusted(issuer);

  auto bucketHash = getUnreviewedTrxBucket();

  ContentWrapper cw(trx_info);

  //TODO: Add check for mandatory fields

  const string& trxSource = cw.getOrFail(DETAILS, UNREVIEWED_TRX_SOURCE)->getAs<string>();

  const string& trxCursor = cw.getOrFail(DETAILS, UNREVIEWED_TRX_CURSOR)->getAs<string>();

  cursor_table cursorsTbl(get_self(), get_self().value);

  auto sourceIdx = cursorsTbl.get_index<"bysource"_n>();
  
  if (auto sourceIdxItr = sourceIdx.find(util::hashString(trxSource)); 
      sourceIdxItr == sourceIdx.end()) {
    cursorsTbl.emplace(get_self(), [&](cursor& c) {
      c.key = cursorsTbl.available_primary_key();
      c.source = trxSource;
      c.last_cursor = trxCursor;
    });
  }
  else {
    sourceIdx.modify(sourceIdxItr, get_self(), [&](cursor& c) {
      c.last_cursor = trxCursor;
    });
  }
  
  Document newUnrvwdTrx(get_self(), issuer, std::move(trx_info));

  Edge(get_self(), issuer, bucketHash, newUnrvwdTrx.getHash(), name(UNREVIEWED_EDGE));
}

ACTION
accounting::setsetting(string setting, Content::FlexValue value)
{
  require_auth(get_self());

  Settings& settings = Settings::instance();
  settings.addOrReplace(setting, std::move(value));
}

ACTION
accounting::remsetting(string setting)
{
  require_auth(get_self());

  Settings& settings = Settings::instance();
  settings.remove(setting);
}

ACTION
accounting::addtrustacnt(name account)
{
  require_auth(get_self());

  EOS_CHECK(is_account(account), "Account must exist before adding it");

  Settings& settings = Settings::instance();
  
  auto cw = settings.getWrapper();

  auto [idx, group] = cw.getGroupOrCreate(TRUSTED_ACCOUNTS_GROUP);

  Content acnt(TRUSTED_ACCOUNT_LABEL, account);

  //Check if account already exists
  if (std::find(group->begin(), group->end(), acnt) == group->end()) {
    //group->push_back(acnt);
    settings.add(acnt.label, acnt.value, TRUSTED_ACCOUNTS_GROUP);
  }
  else {
    EOS_CHECK(false, "Account is trusted already");
  }
}

ACTION
accounting::remtrustacnt(name account)
{
  require_auth(get_self());

  Settings& settings = Settings::instance();
  settings.remove(Content(TRUSTED_ACCOUNT_LABEL, account), 
                  TRUSTED_ACCOUNTS_GROUP);
}

ACTION 
accounting::clearunrvwd(int64_t max_removable_trx)
{
  require_auth(get_self());
  
  std::pair<bool, Edge> edgePair;

  //int64_t maxToRemove = MAX_REMOVABLE_DOCS;
  int64_t maxToRemove = max_removable_trx;

  while ((edgePair = Edge::getIfExists(get_self(), getUnreviewedTrxBucket(), name(UNREVIEWED_EDGE)),
         edgePair.first) && maxToRemove--) {
    Edge& edge = edgePair.second;
    //Erase document and its edges
    m_documentGraph.eraseDocument(edge.to_node, true);
  }

  cursor_table cursorsTbl(get_self(), get_self().value);

  auto it = cursorsTbl.begin();

  maxToRemove = max_removable_trx;

  while (it != cursorsTbl.end() && maxToRemove--) {
    it = cursorsTbl.erase(it);
  }
}

/**
* Group Label: details
* documents: int
* edges: int
* exchange_rates: int
* currencies: int
* cursors: int
*/
ACTION
accounting::clean(ContentGroups& tables) 
{
  require_auth(get_self());

  ContentWrapper cw(tables);

  if (cw.getOrFail(DETAILS, "documents")->getAs<int64_t>() == 1) {
    util::cleanuptable<Document::document_table>(get_self());
  }

  if (cw.getOrFail(DETAILS, "edges")->getAs<int64_t>() == 1) {
    util::cleanuptable<Edge::edge_table>(get_self());
  }

  if (cw.getOrFail(DETAILS, "exchange_rates")->getAs<int64_t>() == 1) {
    util::cleanuptable<exchange_rates_table>(get_self()); 
  }

  if (cw.getOrFail(DETAILS, "currencies")->getAs<int64_t>() == 1) {
    util::cleanuptable<currencies_table>(get_self()); 
  }

  if (cw.getOrFail(DETAILS, "cursors")->getAs<int64_t>() == 1) {
    util::cleanuptable<cursor_table>(get_self()); 
  }

  if (cw.getOrFail(DETAILS, "unreviewedtrx")->getAs<int64_t>() == 1) {
    eosio::action(
      eosio::permission_level{get_self(), "active"_n},
      get_self(),
      "clearunrvwd"_n,
      std::make_tuple(int64_t(100))
    ).send();
  }
}

void
accounting::requireTrusted(name account)
{
  Settings& settings = Settings::instance();
  
  auto cw = settings.getWrapper();

  if (auto [idx, group] = cw.getGroup(TRUSTED_ACCOUNTS_GROUP); group) {
    
    auto isSameAccnt =  [account](const Content& ctn) {
      return ctn.getAs<name>() == account;
    };

    //Warning: I'm assuming that content_group_label is always the first
    //item so I skip it
    auto it = std::find_if(group->begin() + 1, group->end(), isSameAccnt);

    if (it != group->end()) { return; }       
  }
  //else no trusted accounts

  EOS_CHECK(false, "Only trusted accounts can perform this action");
}

checksum256 
accounting::getUnreviewedTrxBucket() 
{
  auto rootHash = getRoot().getHash();
  name edgeName = name(UNREVIEWED_BUCKET_EDGE);
  
  //Check if we already have the bucket
  auto edges = m_documentGraph.getEdgesFrom(rootHash, edgeName);
  
  //Create if it doesn't exit
  if (edges.empty()) {
    Document unreviewedTrxBucket = Document(get_self(), get_self(), ContentGroups{
      ContentGroup{
        Content{CONTENT_GROUP_LABEL, DETAILS}
      },
      ContentGroup{
        Content{CONTENT_GROUP_LABEL, SYSTEM},
        Content{NAME_LABEL, UNREVIEWED_BUCKET_LABEL},
        Content{TYPE_LABEL, UNREVIEWED_EDGE}
      },
    });
    Edge(get_self(), get_self(), rootHash, unreviewedTrxBucket.getHash(), edgeName);

    return unreviewedTrxBucket.getHash();
  }
  else {
    EOS_CHECK(edges.size() == 1, "There are more than 1 unreviewed transactions bucket");

    return edges[0].to_node;
  }  
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

  return {
    details,
    getSystemGroup("opening_balances", "account"),
    /*, openings*/
  };
}

ContentGroups
accounting::getEquityAccount(checksum256 parent)
{
  ContentGroup details {
    Content(CONTENT_GROUP_LABEL, DETAILS),
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

  return {
    details,
    getSystemGroup("equity", "account"),
    /*, openings*/
  };
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

ContentGroup
accounting::getTrxComponent(checksum256 account, 
                            string memo, 
                            asset amount, 
                            string label)
{
  return {
    Content{CONTENT_GROUP_LABEL, std::move(label)},
    Content{COMPONENT_ACCOUNT, account},
    Content{COMPONENT_DATE, current_time_point()},
    Content{COMPONENT_MEMO, memo},
    Content{COMPONENT_AMMOUNT, amount}
  };
}

void 
accounting::parent(name creator, 
                   checksum256 parent, 
                   checksum256 child, 
                   string_view fromToEdge, 
                   string_view toFromEdge)
{
  Edge(get_self(), creator, parent, child, name(fromToEdge));
  Edge(get_self(), creator, child, parent, name(toFromEdge));
}

static ContentGroups
getRootContentGroups() {
  return {
    ContentGroup {
      Content{CONTENT_GROUP_LABEL, DETAILS},
      Content{ROOT_NODE, accounting::getName()},
    },
    accounting::getSystemGroup("root", ROOT_NODE)
  };
}

ContentGroup
accounting::getSystemGroup(const char* nodeName, const char* type)
{
  return {
    Content(CONTENT_GROUP_LABEL, SYSTEM),
    Content(NAME_LABEL, nodeName),
    Content(TYPE_LABEL, type)
  };
}


const Document& 
accounting::getRoot()
{
  //This assumes the root document won't change
  static Document rootDoc = Document::getOrNew(getName(), 
                                               getName(),
                                               getRootContentGroups());
  
  return rootDoc;
}

name
accounting::getName() 
{
  return g_contractName;
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