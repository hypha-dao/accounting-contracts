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

  // Transaction might be empty
  EOS_CHECK(
    trxInfo.size() >= 2,
    "Transaction must contain at least 1 component"
  );

  // Extract the components
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

    if (auto [idx, c] = trx.get(i, COMPONENT_FROM); c) {  
      component.from = c->getAs<string>();
    }

    if (auto [idx, c] = trx.getOrFail(i, COMPONENT_TO); c) {
      component.to = c->getAs<string>();
    }

    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_TAG_TYPE, "Missing 'type' content on trx component");
      component.type = c->getAs<string>();
      EOS_CHECK(
        component.type == DEBIT_TAG_TYPE || component.type == CREDIT_TAG_TYPE,
        util::to_str("Invalid component type:", component.type, " expected [", DEBIT_TAG_TYPE, " or ", CREDIT_TAG_TYPE, "]")
      );
    }

    if (auto [idx, c] = trx.getOrFail(i, COMPONENT_AMMOUNT); c) {

      component.amount = c->getAs<asset>();

      EOS_CHECK(component.amount.is_valid(), "Not valid asset: " + 
                component.amount.to_string() + " at " + 
                std::to_string(i) + 
                " memo:" + component.memo);
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
  type = cw.getOrFail(detailsIdx, COMPONENT_TAG_TYPE).second->getAs<std::string>();
}

static int64_t 
getSign(int64_t v) 
{
  return v ? static_cast<int64_t>(v > 0) - static_cast<int64_t>(v < 0) : 0;
}

void
Transaction::checkBalanced()
{
  TRACE_FUNCTION()

  std::map<std::string, asset> totals;

  for (Component & component : m_components) {
    asset quantity = component.amount;
    std::string code = quantity.symbol.code().to_string();

    if (component.type == CREDIT_TAG_TYPE) {
      quantity.set_amount(quantity.amount * -1);
    }
    
    auto titr = totals.find(code);
    if (titr == totals.end()) {
      totals.insert(make_pair(code, quantity));
    }
    else {
      totals.at(code) = util::addAssetsAdjustingPrecision(titr->second, quantity);
    }
  }

  int64_t zero = 0;

  for (auto itr = totals.begin(); itr != totals.end(); itr++) {
    EOS_CHECK(
      itr->second.amount == zero,
      util::to_str("Transaction is unbalanced. Asset ", itr->first, " sums up to ", itr->second)
    )
  }
}

}