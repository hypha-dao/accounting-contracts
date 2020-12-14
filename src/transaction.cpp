#include "transaction.hpp"

#include "constants.hpp"

namespace hypha {

using eosio::check;

Transaction::Transaction(ContentGroups trxInfo)
{
  ContentWrapper trx(trxInfo);

  auto [hIdx, headerGroup] = trx.getGroup("header");

  check(headerGroup, "Missing 'header' group in transaction document");

  //Check all the header fields are present
  memo = std::move(trx.getOrFail(hIdx, TRX_MEMO).second->getAs<string>());
  date = trx.getOrFail(hIdx, TRX_DATE).second->getAs<time_point>();
  ledger = trx.getOrFail(hIdx, TRX_LEDGER).second->getAs<checksum256>();

  //Extract the components
  for (size_t i = 0; i < trxInfo.size(); ++i) {
    if (i == hIdx) { continue; };

    auto groupLabel = trx.getGroupLabel(i);

    check(!groupLabel.empty(), "Unexpected content group withouh label: " + 
                                std::to_string(i));

    check(groupLabel == "component", "Wrong content_group_label value [" + 
                                      string(groupLabel) + 
                                      "] expecting 'component'");

    Component component;
    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_AMMOUNT, "Missing amount content on trx component");
      component.amount = c->getAs<asset>();
    }
    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_MEMO, "Missing memo content on trx component");
      component.memo = std::move(c->getAs<string>());
    }
    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_ACCOUNT, "Missing account content on trx component");
      component.account = c->getAs<checksum256>();
    }
    components.emplace_back(std::move(component));
  }  
}

bool Transaction::isBalanced()
{
  for (const auto& component : components) {
    
  }

  return false;
}

}