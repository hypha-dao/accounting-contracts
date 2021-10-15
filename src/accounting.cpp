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
accounting::createroot(std::string)
{
  TRACE_FUNCTION()

  require_auth(get_self());
  getRoot();
}

ACTION
accounting::addledger(name creator, ContentGroups& ledger_info)
{
  TRACE_FUNCTION()

  require_auth(creator);
  requireTrusted(creator);

  ContentWrapper cw(ledger_info);

  auto& ledgerName = cw.getOrFail(DETAILS, "name")->getAs<string>();
  
  ledger_info.push_back(getSystemGroup(ledgerName.c_str(), "ledger"));

  Document ledger(get_self(), creator, std::move(ledger_info));

  Document trxBucket(get_self(), creator, {
    ContentGroup{
      Content {CONTENT_GROUP_LABEL, DETAILS},
      Content {CREATE_DATE, current_time_point()},
    },
    getSystemGroup(TRX_BUCKET_LABEL, TRX_BUCKET_EDGE)
  });

  Edge(get_self(), creator, ledger.getHash(), trxBucket.getHash(), name(TRX_BUCKET_EDGE));

  Edge(get_self(), creator, getRoot().getHash(), ledger.getHash(), name("ledger"));
}

ACTION
accounting::createacc(name creator, ContentGroups& account_info)
{ 
  TRACE_FUNCTION()

  require_auth(creator);
  requireTrusted(creator);

  ContentWrapper contentWrap(account_info);

  auto [dIdx, details] = contentWrap.getGroup(DETAILS);

  EOS_CHECK(details, "Details group was expected but not found in account info");
  
  auto parentHash = contentWrap.getOrFail(dIdx, PARENT_ACCOUNT).second->getAs<checksum256>();

  auto accountType = contentWrap.getOrFail(dIdx, ACCOUNT_TYPE).second->getAs<int64_t>();
  
  std::string accountTagType = "";

  if (auto [tagIdx, accTagType] = contentWrap.get(dIdx, ACCOUNT_TAG_TYPE); 
      accTagType) {
    accountTagType = accTagType->getAs<std::string>();
  }

  auto accountCode = contentWrap.getOrFail(dIdx, ACCOUNT_CODE).second->getAs<string>();

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
      Content{ACCOUNT_TAG_TYPE, accountTagType},
      Content{ACCOUNT_CODE, accountCode},
      // Content{PARENT_ACCOUNT, parentHash},
      Content{ACCOUNT_PATH, getAccountPath(accountName, parentHash, ledger)}
    },
    getSystemGroup(accountName.c_str(), "account"),
  });

  auto& settings = Settings::instance();
  int64_t nextBalanceID = settings.getOrDefault("next_balances_id", int64_t{0});
  settings.addOrReplace("next_balances_id", nextBalanceID + 1);

  auto balanceSystemGroup = getBalancesSystemGroup(nextBalanceID);

  //Create balances document
  Document balances(get_self(), creator, {
    ContentGroup{
      Content{CONTENT_GROUP_LABEL, BALANCES},
    },
    std::move(balanceSystemGroup)
  });

  Edge(get_self(), creator, account.getHash(), balances.getHash(), name(BALANCES));
  
  parent(creator, parentHash, account.getHash());
}

ACTION
accounting::upserttrx(const name & issuer, const checksum256 & trx_hash, ContentGroups & trx_info, bool approve)
{
  TRACE_FUNCTION()

  require_auth(issuer);
  requireTrusted(issuer);

  checksum256 nullHash;
  
  if (trx_hash == nullHash) {
    createTransaction(issuer, uint64_t(0), trx_info, approve);
  } else {
    EOS_CHECK(
      !isApproved(trx_hash),
      util::to_str("Cannot modify an approved transaction: ", trx_hash)
    )

    Document trxDoc(get_self(), trx_hash);
    ContentWrapper cw = trxDoc.getContentWrapper();

    int64_t trxId = cw.getOrFail(DETAILS, TRX_ID)->getAs<int64_t>();

    deleteTransaction(trx_hash);
    createTransaction(issuer, trxId, trx_info, approve);
  }
}

ACTION
accounting::deletetrx(const name & deleter, const checksum256 & trx_hash) 
{
  TRACE_FUNCTION()

  require_auth(deleter);
  requireTrusted(deleter);

  deleteTransaction(trx_hash);
}

ACTION
accounting::balancetrx(const name & issuer, checksum256 & trx_hash)
{
  TRACE_FUNCTION()

  require_auth(issuer);
  requireTrusted(issuer);

  auto unapprovedEdge = m_documentGraph.getEdgesFrom(trx_hash, name(UNAPPROVED_TRX));

  EOS_CHECK(
    !unapprovedEdge.empty(),
    util::to_str("Cannot balance approved transaction: ", trx_hash)
  )

  auto trxComponents = m_documentGraph.getEdgesFrom(trx_hash, name(COMPONENT_TYPE));
  for (Edge& componentEdge : trxComponents) {
    Document cmpDoc(get_self(), componentEdge.getToNode());
    //Verify component has linked account
    Edge::get(get_self(), cmpDoc.getHash(), name(ACCOUNT_EDGE));
  }

  Document trxDoc(get_self(), trx_hash);

  auto trxCW = trxDoc.getContentWrapper();

  auto detailsGroup = trxCW.getGroupOrFail(DETAILS);

  ContentWrapper::insertOrReplace(*detailsGroup, Content{TRX_APPROVER, issuer});

  trxDoc = m_documentGraph.updateDocument(issuer, trxDoc.getHash(), trxDoc.getContentGroups());

  trx_hash = trxDoc.getHash();

  Transaction trx(trxDoc, m_documentGraph);
  
  trx.checkBalanced();

  for (auto & component : trx.getComponents()) {
    changeAcctBalanceRecursively(
      component.account, 
      trx.getLedger(), 
      ((component.type == CREDIT_TAG_TYPE) ? -1 : 1) * component.amount, 
      false
    );
  }

  auto ledgerToTrxBucket = Edge::get(get_self(), trx.getLedger(), name(TRX_BUCKET_EDGE));
  auto bucketHash = ledgerToTrxBucket.getToNode();
  
  Edge::get(get_self(), bucketHash, trx_hash, name(UNAPPROVED_TRX)).erase();
  Edge::get(get_self(), trx_hash, bucketHash, name(UNAPPROVED_TRX)).erase();

  parent(issuer, bucketHash, trx_hash, APPROVED_TRX, APPROVED_TRX);
}

void
accounting::createTransaction(const name & issuer, int64_t trxId, ContentGroups & trx_info, bool approve)
{
  TRACE_FUNCTION()

  if (trxId == 0) {
    Settings& settings = Settings::instance();
    trxId = settings.getOrDefault("next_trx_id", int64_t(1));
    settings.addOrReplace("next_trx_id", trxId + 1);
  }

  ContentWrapper cw(trx_info);
  ContentGroup & detailsGroup = *cw.getGroupOrFail(DETAILS);

  ContentWrapper::insertOrReplace(detailsGroup, Content{ TRX_ID, trxId });

  Transaction trx(trx_info);

  if (approve) {
    trx.checkBalanced();

    ContentWrapper::insertOrReplace(detailsGroup, Content{ TRX_APPROVER, issuer });

    for (auto & component : trx.getComponents()) {
      changeAcctBalanceRecursively(
        component.account, 
        trx.getLedger(), 
        ((component.type == CREDIT_TAG_TYPE) ? -1 : 1) * component.amount, 
        false
      );
    }
  }

  Document trxDoc(get_self(), issuer, { detailsGroup, getSystemGroup(TRX_LABEL, TRX_TYPE) });

  const std::vector<uint64_t> & allowed_currencies = getAllowedCurrencies();

  for (auto & compnt : trx.getComponents()) {

    EOS_CHECK(
      compnt.amount.amount >= 0,
      "Component amount must be a positive quantity."
    )

    EOS_CHECK(
      isAllowedCurrency(compnt.amount.symbol, allowed_currencies),
      util::to_str("Currency ", compnt.amount.symbol.code(), " is not allowed.")
    )
    
    Document compntDoc(get_self(), issuer, {
      getTrxComponent(compnt.account, compnt.memo, compnt.amount, compnt.from, compnt.to, compnt.type, DETAILS),
      getSystemGroup(COMPONENT_LABEL, COMPONENT_TYPE) 
    });

    parent(issuer, trxDoc.getHash(), compntDoc.getHash(), COMPONENT_TYPE, TRX_TYPE);

    if (compnt.event) {
      bindevent(issuer, *compnt.event, compntDoc.getHash());
    }

    Edge compntToAcc(get_self(), issuer, compntDoc.getHash(), compnt.account, name(ACCOUNT_EDGE));
  }

  auto ledgerToTrxBucket = Edge::get(get_self(), trx.getLedger(), name(TRX_BUCKET_EDGE));
  auto bucketHash = ledgerToTrxBucket.getToNode();

  std::string edgeName = approve ? APPROVED_TRX : UNAPPROVED_TRX;

  parent(issuer, bucketHash, trxDoc.getHash(), edgeName, edgeName);
}

void
accounting::deleteTransaction(const checksum256 & trx_hash)
{
  TRACE_FUNCTION()

  EOS_CHECK(
    !isApproved(trx_hash),
    util::to_str("Cannot delete an approved transaction: ", trx_hash)
  )

  Document trxDoc(get_self(), trx_hash);

  Transaction trx(trxDoc, m_documentGraph);

  for (auto & component : trx.getComponents()) {
    m_documentGraph.eraseDocument(*component.hash, true);
  }

  m_documentGraph.eraseDocument(trx_hash, true);
}

bool
accounting::isApproved(const checksum256 & trx_hash)
{
  auto unapprovedEdge = m_documentGraph.getEdgesFrom(trx_hash, name(UNAPPROVED_TRX));
  return unapprovedEdge.empty();
}

ACTION
accounting::updateacc(name updater, checksum256 account_hash, ContentGroups account_info) 
{
  require_auth(updater);
  requireTrusted(updater);

  Document accountDoc(get_self(), account_hash);

  ContentWrapper accountCW = accountDoc.getContentWrapper();

  EOS_CHECK(
    accountCW.getOrFail(SYSTEM, TYPE_LABEL)->getAs<std::string>() == "account",
    "Expected document type of 'account'"
  )

  auto newInfoCW = ContentWrapper(account_info);

  auto [newDetailsIdx, newDetails] = newInfoCW.getGroup(DETAILS);

  EOS_CHECK(newDetails != nullptr, "Details group was expected but not found in account_info");
 
  auto accountType = newInfoCW.getOrFail(newDetailsIdx, ACCOUNT_TYPE).second->getAs<int64_t>();
  
  auto accountTagType = newInfoCW.getOrFail(newDetailsIdx, ACCOUNT_TAG_TYPE).second->getAs<string>();

  auto accountCode = newInfoCW.getOrFail(newDetailsIdx, ACCOUNT_CODE).second->getAs<string>();

  auto accountName = newInfoCW.getOrFail(newDetailsIdx, ACCOUNT_NAME).second->getAs<string>();
  
  //Create the account
  auto oldAccountName = accountCW.getOrFail(DETAILS, ACCOUNT_NAME)->getAs<std::string>();

  auto oldPath = accountCW.getOrFail(DETAILS, ACCOUNT_PATH)->getAs<std::string>();

  EOS_CHECK(
    oldAccountName == accountName,
    "Changing account name is not supported"
  )

  auto newAccountDetails = ContentGroup{
    Content{CONTENT_GROUP_LABEL, DETAILS},
    Content{ACCOUNT_NAME, accountName},
    Content{ACCOUNT_TYPE, accountType},
    Content{ACCOUNT_TAG_TYPE, accountTagType},
    Content{ACCOUNT_CODE, accountCode},
    Content{ACCOUNT_PATH, oldPath}
  };
  
  *accountCW.getGroupOrFail(DETAILS) = newAccountDetails;
  
  m_documentGraph.updateDocument(updater, accountDoc.getHash(), accountDoc.getContentGroups());

  //In order to support changing name, we must also store ledger or find it
  //by navigating all the parent accounts
  //Also we need to update children nodes path names

  // if (oldAccountName != accountName) {
  //   auto parentHash = Edge::get(get_self(), accountDoc.getHash(), name(OWNED_BY)).getToNode();

  //   auto accountEdges = m_documentGraph.getEdgesFrom(parentHash, name("account"));

  //   TableWrapper<document_table> docs(get_self(), get_self().value);

  //   EOS_CHECK(docs.contains_by<"idhash"_n>(parentHash), 
  //         "The parent document doesn't exists: " + readableHash(parentHash));
    
  //   //Check if there isn't already an account with the same name
  //   for (const auto& edge : accountEdges) {
      
  //     auto account = docs.get_by<"idhash"_n>(edge.to_node);

  //     ContentWrapper cg(account.getContentGroups());

  //     auto name = cg.getOrFail(DETAILS, ACCOUNT_NAME)->getAs<string>();

  //     EOS_CHECK(
  //       util::toLowerCase(std::move(name)) != util::toLowerCase(accountName), 
  //       "There is already an account with name: " + accountName
  //     );
  //   }
  // }

}

ACTION
accounting::newevent(name issuer, ContentGroups trx_info) 
{
  TRACE_FUNCTION()
  
  require_auth(issuer);
  requireTrusted(issuer);

  auto bucketHash = getEventBucket();

  ContentWrapper cw(trx_info);

  //TODO: Add check for mandatory fields

  const string& trxSource = cw.getOrFail(DETAILS, EVENT_SOURCE)->getAs<string>();

  const string& trxCursor = cw.getOrFail(DETAILS, EVENT_CURSOR)->getAs<string>();

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
  
  Document newevent(get_self(), issuer, std::move(trx_info));

  Edge(get_self(), issuer, bucketHash, newevent.getHash(), name(EVENT_EDGE));
}

ACTION
accounting::bindevent(name updater, checksum256 event_hash, checksum256 component_hash)
{
  TRACE_FUNCTION()
  
  requireTrusted(updater);
  require_auth(updater);

  EOS_CHECK(
    m_documentGraph.getEdgesFrom(event_hash, name(COMPONENT_TYPE)).empty(),
    util::to_str("Event: ", event_hash, " is already binded to a component")
  )

  EOS_CHECK(
    m_documentGraph.getEdgesFrom(component_hash, name(EVENT_EDGE)).empty(),
    util::to_str("Component: ", component_hash, " is already binded to an event")
  )

  parent(updater, event_hash, component_hash, COMPONENT_TYPE, EVENT_EDGE);
}

ACTION
accounting::unbindevent(name updater, checksum256 event_hash, checksum256 component_hash)
{
  TRACE_FUNCTION()
  
  require_auth(updater);
  requireTrusted(updater);

  auto trxEdge = m_documentGraph.getEdgesFrom(component_hash, name(TRX_TYPE));

  EOS_CHECK(
    trxEdge.size() == 1,
    util::to_str("Missing transaction edge from component: ", component_hash)
  )

  checksum256 trxHash = trxEdge[0].getToNode();
  //TODO: Verify if the component's transaction hasn't been approved yet

  auto unapprovedEdge = m_documentGraph.getEdgesFrom(trxHash, name(UNAPPROVED_TRX));

  EOS_CHECK(
    !unapprovedEdge.empty(),
    util::to_str("Cannot unbind event from approved transaction: ", trxHash)
  )

  Edge::get(get_self(), event_hash, component_hash, name(COMPONENT_TYPE)).erase();
  Edge::get(get_self(), component_hash, event_hash, name(EVENT_EDGE)).erase();
}

ACTION
accounting::setsetting(string setting, Content::FlexValue value)
{
  TRACE_FUNCTION()

  require_auth(get_self());

  Settings& settings = Settings::instance();
  settings.addOrReplace(setting, std::move(value));
}

ACTION
accounting::remsetting(string setting)
{
  TRACE_FUNCTION()

  require_auth(get_self());

  Settings& settings = Settings::instance();
  settings.remove(setting);
}

ACTION
accounting::addtrustacnt(name account)
{
  TRACE_FUNCTION()

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
  TRACE_FUNCTION()

  require_auth(get_self());

  Settings& settings = Settings::instance();
  settings.remove(Content(TRUSTED_ACCOUNT_LABEL, account), 
                  TRUSTED_ACCOUNTS_GROUP);
}

ACTION
accounting::addcurrency(symbol & currency_symbol)
{
  TRACE_FUNCTION()

  require_auth(get_self());

  Settings & settings = Settings::instance();

  auto cw = settings.getWrapper();
  auto [idx, group] = cw.getGroupOrCreate(ALLOWED_CURRENCIES_GROUP);

  for (auto itr = group->begin(); itr != group->end(); itr++) {
    if (itr->label == ALLOWED_CURRENCIES_LABEL) {
      EOS_CHECK(
        (itr->getAs<asset>()).symbol.code().raw() != currency_symbol.code().raw(),
        "Currency symbol already exists."
      )
    }
  }

  settings.add(ALLOWED_CURRENCIES_LABEL, asset(0, currency_symbol), ALLOWED_CURRENCIES_GROUP);
}

ACTION
accounting::remcurrency(symbol & currency_symbol)
{
  TRACE_FUNCTION()

  require_auth(get_self());

  Settings & settings = Settings::instance();
  settings.remove(Content(ALLOWED_CURRENCIES_LABEL, asset(0, currency_symbol)), ALLOWED_CURRENCIES_GROUP);
}

ACTION 
accounting::clearevent(int64_t max_removable_trx)
{
  TRACE_FUNCTION()

  require_auth(get_self());
  
  std::pair<bool, Edge> edgePair;

  //int64_t maxToRemove = MAX_REMOVABLE_DOCS;
  int64_t maxToRemove = max_removable_trx;

  while ((edgePair = Edge::getIfExists(get_self(), getEventBucket(), name(EVENT_EDGE)),
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
  TRACE_FUNCTION()

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

  if (cw.getOrFail(DETAILS, "events")->getAs<int64_t>() == 1) {
    eosio::action(
      eosio::permission_level{get_self(), "active"_n},
      get_self(),
      "clearevent"_n,
      std::make_tuple(int64_t(100))
    ).send();
  }
}

void
accounting::requireTrusted(name account)
{
  TRACE_FUNCTION()

  Settings& settings = Settings::instance();
  
  auto cw = settings.getWrapper();

  if (auto [idx, group] = cw.getGroup(TRUSTED_ACCOUNTS_GROUP); group) {
    
    auto isSameAccnt =  [account](const Content& ctn) {
      return ctn.getAs<name>() == account;
    };

    //WARNING: I'm assuming that content_group_label is always the first
    //item so I skip it
    auto it = std::find_if(group->begin() + 1, group->end(), isSameAccnt);

    if (it != group->end()) { return; }       
  }
  //else no trusted accounts

  EOS_CHECK(false, "Only trusted accounts can perform this action");
}

checksum256 
accounting::getEventBucket() 
{
  TRACE_FUNCTION()
  
  auto rootHash = getRoot().getHash();
  name edgeName = name(EVENT_BUCKET_EDGE);
  
  //Check if we already have the bucket
  auto edges = m_documentGraph.getEdgesFrom(rootHash, edgeName);
  
  //Create if it doesn't exit
  if (edges.empty()) {
    Document eventBucket = Document(get_self(), get_self(), ContentGroups{
      ContentGroup{
        Content{CONTENT_GROUP_LABEL, DETAILS}
      },
      getSystemGroup(EVENT_BUCKET_LABEL, EVENT_EDGE)
    });
    Edge(get_self(), get_self(), rootHash, eventBucket.getHash(), edgeName);

    return eventBucket.getHash();
  }
  else {
    EOS_CHECK(edges.size() == 1, "There are more than 1 events bucket");

    return edges[0].to_node;
  }  
}

std::string 
accounting::getAccountPath(std::string account, checksum256 parent, const checksum256& ledger) 
{
  TRACE_FUNCTION()

  const char* SEPARATOR = " > ";

  TableWrapper<document_table> docs(get_self(), get_self().value);

  while (parent != ledger) {
    auto doc = docs.get_by<"idhash"_n>(parent);

    ContentWrapper accWrapper = doc.getContentWrapper();

    const string& accountName = accWrapper.getOrFail(DETAILS, ACCOUNT_NAME)->getAs<std::string>();

    account = util::to_str(accountName, SEPARATOR, account);

    auto toParentEdges = m_documentGraph.getEdgesFrom(doc.getHash(), name(OWNED_BY));

    EOS_CHECK(
      !toParentEdges.empty(),
      util::to_str("Missing edge [ownedby] from account: ",  doc.getHash(), " to parent")
    )

    parent = toParentEdges[0].getToNode();
  }

  return account;
}

ContentGroup
accounting::getTrxComponent(checksum256 account, 
                            string memo, 
                            asset amount, 
                            string from, 
                            string to,
                            string type,
                            string label)
{
  return {
    Content{CONTENT_GROUP_LABEL, std::move(label)},
    Content{COMPONENT_ACCOUNT, account},
    Content{COMPONENT_DATE, current_time_point()},
    Content{COMPONENT_MEMO, memo},
    Content{COMPONENT_FROM, from},
    Content{COMPONENT_TO, to},
    Content{COMPONENT_TAG_TYPE, type},
    Content{COMPONENT_AMMOUNT, amount}
  };
}

ContentGroup 
accounting::getBalancesSystemGroup(int64_t id) 
{
  auto systemGroup = getSystemGroup(BALANCES, 
                                    BALANCES);

  systemGroup.push_back(Content{CREATE_DATE, current_time_point()});
  systemGroup.push_back(Content{"balance_id", id});

  return systemGroup;
}

void 
accounting::parent(name creator, 
                   checksum256 parent, 
                   checksum256 child, 
                   string_view fromToEdge, 
                   string_view toFromEdge)
{
  TRACE_FUNCTION()

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
accounting::getSystemGroup(std::string nodeName, std::string type)
{
  //TODO: Add contract version
  return {
    Content(CONTENT_GROUP_LABEL, SYSTEM),
    Content(NAME_LABEL, std::move(nodeName)),
    Content(TYPE_LABEL, std::move(type))
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

Document
accounting::getAccountBalances(checksum256 account)
{
  return Document(get_self(),
                  Edge::get(get_self(), account, name(BALANCES)).getToNode());
}

void
accounting::changeAcctBalanceRecursively(
  const checksum256 & account, 
  const checksum256 & ledger, 
  const asset & quantity,
  const bool onlyGlobal
)
{
  TRACE_FUNCTION()

  if (account == ledger) return;

  auto balancesDoc = getAccountBalances(account);
  auto balancesCW = balancesDoc.getContentWrapper();

  auto [groupIdx, balancesGroup] = balancesCW.getGroup(BALANCES);

  EOS_CHECK(
    balancesGroup != nullptr,
    util::to_str("Missing balances group from balance document:", balancesDoc.getHash())
  )

  if (!onlyGlobal) {
    std::string balanceLabel = std::string("account_") + quantity.symbol.code().to_string();
    asset newAssetBalance;

    if (auto [_, item] = balancesCW.get(groupIdx, balanceLabel); item) {
      auto assetBalance = item->getAs<asset>();
      newAssetBalance = util::addAssetsAdjustingPrecision(assetBalance, quantity);
    }
    else {
      newAssetBalance = quantity;
    }

    ContentWrapper::insertOrReplace(*balancesGroup, Content{ balanceLabel, newAssetBalance });
  }

  std::string balanceLabel = std::string("global_") + quantity.symbol.code().to_string();
  asset newAssetBalance;

  if (auto [_, item] = balancesCW.get(groupIdx, balanceLabel); item) {
    auto assetBalance = item->getAs<asset>();
    newAssetBalance = util::addAssetsAdjustingPrecision(assetBalance, quantity);
  }
  else {
    newAssetBalance = quantity;
  }

  ContentWrapper::insertOrReplace(*balancesGroup, Content{ balanceLabel, newAssetBalance });

  m_documentGraph.updateDocument(get_self(), balancesDoc.getHash(), balancesCW.getContentGroups());

  auto parentHash = Edge::get(get_self(), account, name(OWNED_BY)).getToNode();

  changeAcctBalanceRecursively(parentHash, ledger, quantity, true);
}

bool
accounting::isAllowedCurrency(const symbol & currency_symbol, const std::vector<uint64_t> & allowed_currencies)
{
  TRACE_FUNCTION()
  return std::binary_search(allowed_currencies.begin(), allowed_currencies.end(), currency_symbol.code().raw());
}

const std::vector<uint64_t>&
accounting::getAllowedCurrencies()
{
  TRACE_FUNCTION()

  static std::vector<uint64_t> allowed_currencies;
  Settings & settings = Settings::instance();

  auto cw = settings.getWrapper();

  if (auto [idx, group] = cw.getGroup(ALLOWED_CURRENCIES_GROUP); group) {
    
    for (auto content = group->begin() + 1; content != group->end(); content++) {
      auto zero_asset = content->getAs<asset>();
      allowed_currencies.push_back(zero_asset.symbol.code().raw());
    }

  }

  EOS_CHECK(allowed_currencies.size() > 0, "There are no allowed currencies.")

  return allowed_currencies;
}

}