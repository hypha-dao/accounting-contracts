#include "transaction.hpp"

#include <map>

#include <document_graph/document_graph.hpp>

#include <accounting.hpp>

#include <logger/logger.hpp>

namespace hypha {

using eosio::check;

Transaction::Transaction(ContentGroups& trxInfo)
{
  TRACE_FUNCTION()

  ContentWrapper trx(trxInfo);

  auto [hIdx, detailsGroup] = trx.getGroup(DETAILS);

  EOS_CHECK(detailsGroup, "Missing 'details' group in transaction document");

  //Check all the details fields are present
  m_memo = trx.getOrFail(hIdx, TRX_MEMO).second->getAs<string>();
  m_name = trx.get(hIdx, TRX_NAME).second->getAs<string>();
  m_date = trx.getOrFail(hIdx, TRX_DATE).second->getAs<time_point>();
  m_ledger = trx.getOrFail(hIdx, TRX_LEDGER).second->getAs<checksum256>();
  m_id = trx.getOrFail(hIdx, TRX_ID).second->getAs<int64_t>();

  //Transaction might be empty
  // EOS_CHECK(
  //   trxInfo.size() >= 2,
  //   "Transaction must contain at least 1 component"
  // );

  //Extract the components
  for (size_t i = 0; i < trxInfo.size(); ++i) {
    if (i == hIdx) { continue; };

    auto groupLabel = trx.getGroupLabel(i);

    EOS_CHECK(!groupLabel.empty(), "Unexpected content group withouh label: " + 
                                std::to_string(i));

    EOS_CHECK(groupLabel == "component", "Wrong content_group_label value [" + 
                                      string(groupLabel) + 
                                      "] expecting 'component'");

    Component component;
    
    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_MEMO, "Missing memo content on trx component");
      component.memo = c->getAs<string>();
    }
    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_ACCOUNT, "Missing account content on trx component");
      component.account = c->getAs<checksum256>();
    }
    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_FROM, "Missing 'from' content on trx component");
      component.from = c->getAs<string>();
    }
    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_TO, "Missing 'to' content on trx component");
      component.to = c->getAs<string>();
    }

    //If there is no ammount, then it should be implied
    if (auto [idx, c] = trx.get(i, COMPONENT_AMMOUNT); c) {
        component.amount = c->getAs<asset>();
        EOS_CHECK(component.amount.is_valid(), "Not valid asset: " + 
                                component.amount.to_string() + " at " + 
                                std::to_string(i) + 
                                " memo:" +component.memo);
    }

    if (auto [idx, event] = trx.get(i, EVENT_EDGE); event) {
      component.event = event->getAs<checksum256>();
    }
    
    m_components.emplace_back(std::move(component));
  }
}

Transaction::Transaction(Document& trxDoc, DocumentGraph& docgraph) 
{
  TRACE_FUNCTION()

  ContentWrapper trx = trxDoc.getContentWrapper();

  auto [hIdx, detailsGroup] = trx.getGroup(DETAILS);

  EOS_CHECK(detailsGroup, "Missing 'details' group in transaction document");

  //Check all the details fields are present
  m_memo = trx.getOrFail(hIdx, TRX_MEMO).second->getAs<string>();
  m_name = trx.getOrFail(hIdx, TRX_NAME).second->getAs<string>();
  m_date = trx.getOrFail(hIdx, TRX_DATE).second->getAs<time_point>();
  m_ledger = trx.getOrFail(hIdx, TRX_LEDGER).second->getAs<checksum256>();
  m_id = trx.getOrFail(hIdx, TRX_ID).second->getAs<int64_t>();

  auto componentEdges = docgraph.getEdgesFrom(trxDoc.getHash(), name(COMPONENT_TYPE));

  for (Edge& componentEdge : componentEdges) {
    const auto cmpHash = componentEdge.getToNode();
    Document cmpDoc(accounting::getName(), cmpHash);
    
    Transaction::Component cmp(cmpDoc.getContentGroups());

    cmp.hash = cmpHash;

    if (auto [exists, eventEdge] = Edge::getIfExists(accounting::getName(), 
                                                     cmpDoc.getHash(), 
                                                     name(EVENT_EDGE)); exists) {
      cmp.event = eventEdge.getToNode();
    }

    m_components.emplace_back(std::move(cmp));
  }
}

Transaction::Component::Component(ContentGroups& data) 
{
  TRACE_FUNCTION()
  ContentWrapper cw(data);

  auto [detailsIdx, detailsGroup] = cw.getGroup(DETAILS);

  EOS_CHECK(
    detailsGroup != nullptr, 
    util::to_str("Missing ", DETAILS, " group")
  )

  account = cw.getOrFail(detailsIdx, COMPONENT_ACCOUNT).second->getAs<checksum256>();
  memo = cw.getOrFail(detailsIdx, COMPONENT_MEMO).second->getAs<std::string>();
  amount = cw.getOrFail(detailsIdx, COMPONENT_AMMOUNT).second->getAs<asset>();
  from = cw.getOrFail(detailsIdx, COMPONENT_FROM).second->getAs<std::string>();
  to = cw.getOrFail(detailsIdx, COMPONENT_TO).second->getAs<std::string>();
}

static int64_t 
getSign(int64_t v) 
{
  return v ? static_cast<int64_t>(v > 0) - static_cast<int64_t>(v < 0) : 0;
}

std::vector<asset>
Transaction::verifyBalanced(DocumentGraph& docgraph)
{
  TRACE_FUNCTION()

  std::map<uint64_t, asset> assetsBySymb;
  std::vector<asset> allAssets;
  
  //Use a copy of the component since we might need to invert the sign of the amount
  for (Component component : m_components) {
    auto& asset = component.amount;

    auto accountEdges = docgraph.getEdgesFromOrFail(*component.hash, name(COMPONENT_ACCOUNT));

    EOS_CHECK(
      accountEdges.size() == 1,
      "There has to exists only 1 account edge from the component"
    )

    auto accountDoc = Document(accounting::getName(), accountEdges[0].getToNode());

    auto accountCW = accountDoc.getContentWrapper();

    auto tagType = accountCW.getOrFail(DETAILS, ACCOUNT_TAG_TYPE)->getAs<string>();

    //Check if the amount belongs to a credit or a debit account
    //if credit we have to negate the amount
    if (tagType == CREDIT_TAG_TYPE) {
      asset.set_amount(asset.amount * -1);
    }
    
    auto [assetIt, inserted] = assetsBySymb.insert({asset.symbol.raw(), asset});  

    auto& accum = assetIt->second;

    //It means the asset existed already
    if (!inserted) { accum.set_amount(accum.amount + asset.amount); }
  }
  
  EOS_CHECK(
    assetsBySymb.size() <= 2,
    "More than 2 currencies were detected"
  );

  std::vector<asset> nonZeroAssets;
  //Only 1 asset is allowed to be implied
  asset* impliedAsset = nullptr;

  for (auto& [sym, asset] : assetsBySymb) {

    //For implied assets, I'm relaying on the fact that it will be the first asset
    //since it's code will be 0 (Invalid) and std::map sorts the data in acending order
    //by default. See accounting.cpp:220
    allAssets.push_back(asset);
    
    if (asset.symbol) {
      if (asset.amount != 0) { nonZeroAssets.push_back(asset); }
    }
    else { //Implied asset
      // EOS_CHECK(!impliedAsset, "Only one component is allowed to be implied: " + asset.to_string());
      // impliedAsset = &asset;
      EOS_CHECK(false, util::to_str("Asset is not valid: ", asset));
    }
  }

  if (impliedAsset) {
    //We assume the pending nonZeroAssets 
    //are going to be implied from this asset
    EOS_CHECK(!nonZeroAssets.empty(), "Implied asset without remaining compnents");

    //There has to be only 1 nonZeroAsset
    EOS_CHECK(nonZeroAssets.size() == 1, "There has to be at most 1 non implied asset");
  }
  else {
    //Balanced
    if (nonZeroAssets.empty()) { return allAssets; }

    //Not valid, there should be at least 2
    EOS_CHECK(nonZeroAssets.size() != 1, "Couldn't balance with remaining posting: " + 
                                     nonZeroAssets.back().to_string());
    
    //If there are 2 components left they
    //must cancel each other
    if (nonZeroAssets.size() == 2) {
      EOS_CHECK(getSign(nonZeroAssets.front().amount) != getSign(nonZeroAssets.back().amount),
            "Remaining assets must cancel each other" + 
            nonZeroAssets.front().to_string() + ", " + nonZeroAssets.back().to_string());
    }
    else {
      EOS_CHECK(false, "Can't balance transaction with remaining components");
    }
  }

  return allAssets;
}

bool 
Transaction::shouldUpdate(Transaction& original) 
{
  if (original.m_name != m_name ||
      original.m_memo != m_memo ||
      original.m_date != m_date) {
    return true;
  }

  return false;
}

}