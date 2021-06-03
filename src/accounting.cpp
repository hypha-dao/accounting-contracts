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
  
  auto accountTagType = contentWrap.getOrFail(dIdx, ACCOUNT_TAG_TYPE).second->getAs<string>();

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

  auto balanceSystemGroup = getSystemGroup(util::to_str(BALANCES), 
                                           BALANCES); 

  balanceSystemGroup.push_back(Content{CREATE_DATE, current_time_point()});

  auto& settings = Settings::instance();
  int64_t nextBalanceID = settings.getOrDefault("next_balances_id", int64_t{0});
  settings.addOrReplace("next_balances_id", nextBalanceID + 1);

  balanceSystemGroup.push_back(Content{CREATE_DATE, current_time_point()});
  balanceSystemGroup.push_back(Content{"balance_id", nextBalanceID});
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

/**
* Transaction structure
* label: details
* transaction_date : time_point
* memo: string
* ledger: chechsum256
* 
* [components]
* label: component
* account: account hash checksum256
* memo: string
* amount: asset
* event?: chechsum256
*  
**/
ACTION
accounting::createtrx(name creator, ContentGroups& trx_info)
{
  TRACE_FUNCTION()

  require_auth(creator);
  requireTrusted(creator);

  Settings& settings = Settings::instance();
  auto nextID = settings.getOrDefault("next_trx_id", int64_t(0));
  settings.addOrReplace("next_trx_id", nextID + 1);

  ContentWrapper cw(trx_info);

  auto detailsGroup = cw.getGroupOrFail(DETAILS);

  ContentWrapper::insertOrReplace(*detailsGroup, Content{TRX_ID, nextID});

  Transaction trx(trx_info);
  
  Document trxDoc(get_self(), creator, { trx.getDetails(), 
                                         getSystemGroup(TRX_LABEL, TRX_TYPE) });

  auto ledgerToTrxBucket = Edge::get(get_self(), trx.getLedger(), name(TRX_BUCKET_EDGE));

  parent(creator, ledgerToTrxBucket.getToNode(), trxDoc.getHash(), UNAPPROVED_TRX, UNAPPROVED_TRX);

  createComponents(trxDoc.getHash(), trx, creator);
}

/**
 Transaction structure
* label: details
* event: checksum256
* ledger: checksum256
*/
ACTION
accounting::createtrxwe(name creator, ContentGroups& trx_info)
{
  TRACE_FUNCTION()

  require_auth(creator);
  requireTrusted(creator);

  Settings& settings = Settings::instance();
  auto nextID = settings.getOrDefault("next_trx_id", int64_t(0));
  settings.addOrReplace("next_trx_id", nextID + 1);

  ContentWrapper cw(trx_info);

  Content* ledger = cw.getOrFail(DETAILS, TRX_LEDGER);

  ContentGroups trxCG {
    ContentGroup {
      Content{CONTENT_GROUP_LABEL, DETAILS},
      Content{TRX_DATE, current_time_point()},
      Content{TRX_ID, nextID},
      Content{TRX_MEMO, ""},
      *ledger
    },
    getSystemGroup(TRX_LABEL, TRX_TYPE)
  };

  Document trxDoc(get_self(), creator, trxCG);

  auto ledgerToTrxBucket = Edge::get(get_self(), ledger->getAs<checksum256>(), name(TRX_BUCKET_EDGE));

  parent(creator, ledgerToTrxBucket.getToNode(), trxDoc.getHash(), UNAPPROVED_TRX, UNAPPROVED_TRX);

  //Create empty component and link it to the event
  checksum256 event = cw.getOrFail(DETAILS, EVENT_EDGE)->getAs<checksum256>();

  Document componentDoc(get_self(), creator, { getTrxComponent(checksum256(), 
                                                             "", 
                                                             asset(), 
                                                             DETAILS),
                                               getSystemGroup(COMPONENT_LABEL, COMPONENT_TYPE)});

  parent(creator, trxDoc.getHash(), componentDoc.getHash(), COMPONENT_TYPE, TRX_TYPE);                       

  eosio::action(
    eosio::permission_level{creator, "active"_n},
    get_self(),
    "bindevent"_n,
    std::make_tuple(creator, event, componentDoc.getHash())
  ).send();
}

ACTION
accounting::updatetrx(name updater, checksum256 trx_hash, ContentGroups& trx_info)
{
  TRACE_FUNCTION()

  require_auth(updater);
  requireTrusted(updater);

  auto unapprovedEdge = m_documentGraph.getEdgesFrom(trx_hash, name(UNAPPROVED_TRX));

  EOS_CHECK(
    !unapprovedEdge.empty(),
    util::to_str("Cannot update approved transaction: ", trx_hash)
  )
  
  //Add new components
  Transaction transaction(trx_info);

  //Verify that trx_hash document 'id' field matches the trx_info 'id'
  {
    Document trxDoc(get_self(), trx_hash);
    int64_t docID = trxDoc.getContentWrapper()
                          .getOrFail(DETAILS, TRX_ID)->getAs<int64_t>();
    EOS_CHECK(
      transaction.getID() == docID,
      util::to_str("The 'id' in trx_info [",
                    transaction.getID(),
                    "] doesn't match the id within document loaded with trx_hash [",
                    trx_hash, ",", docID, "]")
    )
  }

  //Delete Components
  auto oldComponentsEdges = m_documentGraph.getEdgesFrom(trx_hash, name(COMPONENT_TYPE));
  for (Edge& componentEdge : oldComponentsEdges) {
    Document cmpDoc(get_self(), componentEdge.getToNode());
    Transaction::Component cmp(cmpDoc.getContentGroups());
    //Verify if component contains a valid asset 
    //(might be empty if created with 'createtrxwe' action)
    if (cmp.amount.is_valid()) {
      addAssetToAccount(cmp.account, -cmp.amount);
      recalculateGlobalBalances(cmp.account, transaction.getLedger());
    } 
    m_documentGraph.eraseDocument(cmpDoc.getHash(), true);
  }
  
  createComponents(trx_hash, transaction, updater);
}

ACTION
accounting::balancetrx(name issuer, checksum256 trx_hash)
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

  std::vector<asset> assets = trx.verifyBalanced();

  if (assets.empty() || assets.size() > 2) {
    EOS_CHECK(
      false, 
      util::to_str("Assets size has to be 2 or 1, actual:", assets.size())
    );
  } 

  auto ledgerToTrxBucket = Edge::get(get_self(), trx.getLedger(), name(TRX_BUCKET_EDGE));
  auto bucketHash = ledgerToTrxBucket.getToNode();
  
  //Should have unapproved edge
  Edge::get(get_self(), bucketHash, trx_hash, name(UNAPPROVED_TRX)).erase();
  Edge::get(get_self(), trx_hash, bucketHash, name(UNAPPROVED_TRX)).erase();

  //Move to approved edge
  parent(issuer, bucketHash, trx_hash, APPROVED_TRX, APPROVED_TRX);
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

void 
accounting::createComponents(checksum256 trx_hash, Transaction& trx, name creator) 
{
  TRACE_FUNCTION()

  for (auto& compnt : trx.getComponents()) {

    Document compntAcct(get_self(), compnt.account);
    
    Document compntDoc(get_self(), creator, { getTrxComponent(compnt.account, 
                                                             compnt.memo, 
                                                             compnt.amount,
                                                             DETAILS),
                                              getSystemGroup(COMPONENT_LABEL, COMPONENT_TYPE) });

    LOG_MESSAGE(util::to_str("Adding asset:", compnt.amount));

    addAssetToAccount(compnt.account, compnt.amount);
    recalculateGlobalBalances(compnt.account, trx.getLedger());

    /** Connections
     *  component --> 'transaction' --> transaction
     *  component <-- 'component'   <-- transaction 
     *  component --> 'account' --> account
     */
    parent(creator, trx_hash, compntDoc.getHash(), COMPONENT_TYPE, TRX_TYPE);

    if (compnt.event) {
      eosio::action(
        eosio::permission_level{creator, "active"_n},
        get_self(),
        "bindevent"_n,
        std::make_tuple(creator, *compnt.event, compntDoc.getHash())
      ).send();
    }

    Edge compntToAcc(get_self(), creator, compntDoc.getHash(), compntAcct.getHash(), name(ACCOUNT_EDGE));
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

std::vector<accounting::Balance>
accounting::getAccountGlobalBalances(checksum256 account)
{
  auto balances = getAccountBalances(account);
  auto balancesCW = balances.getContentWrapper();

  std::vector<Balance> globalBals;

  auto balancesGroup = balancesCW.getGroupOrFail(BALANCES);

  for (auto& content : *balancesGroup) {
    if (content.label != CONTENT_GROUP_LABEL && 
        util::containsPrefix(content.label, "global_")) {
      globalBals.push_back({content.label, content.getAs<asset>()});
    }
  }

  return globalBals;
}

void 
accounting::addAssetToAccount(checksum256 account, asset amount)
{
  TRACE_FUNCTION()

  auto balancesDoc = getAccountBalances(account);

  addToBalance(balancesDoc, {
    { util::to_str("account_", amount.symbol.code()), amount },
    { util::to_str("global_", amount.symbol.code()), amount },
  });
}

void
accounting::addToBalance(Document& balancesDoc, 
                         const std::vector<Balance>& balances)
{ 
  auto balancesCW = balancesDoc.getContentWrapper();

  auto [groupIdx, balancesGroup] = balancesCW.getGroup(BALANCES);

  EOS_CHECK(
    balancesGroup != nullptr,
    util::to_str("Missing balances group from balance document:", balancesDoc.getHash())
  )

  for (auto& balance: balances) {
    
    asset newAssetBalance;

    if (auto [_, item] = balancesCW.get(groupIdx, balance.label); item) {
      auto assetBalance = item->getAs<asset>();
      newAssetBalance = assetBalance + balance.amount;   
    }
    else {
      newAssetBalance = balance.amount;
    }

    ContentWrapper::insertOrReplace(*balancesGroup, Content{balance.label, newAssetBalance});
  }

  m_documentGraph.updateDocument(get_self(), 
                                 balancesDoc.getHash(), 
                                 std::move(balancesCW.getContentGroups()));
}

void
accounting::recalculateGlobalBalances(checksum256 account, checksum256 ledger)
{
  TRACE_FUNCTION()

  if (account == ledger) {
    return;
  }

  auto accountBalances = getAccountBalances(account);

  auto childrenAccEdges = m_documentGraph.getEdgesFrom(account, name(ACCOUNT_EDGE));

  for (auto& accountEdge: childrenAccEdges) {
    auto accGlobalBalances = getAccountGlobalBalances(accountEdge.getToNode());
    addToBalance(accountBalances, accGlobalBalances);
  }

  auto parentHash = Edge::get(get_self(), account, name(OWNED_BY)).getToNode();

  recalculateGlobalBalances(parentHash, ledger);
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