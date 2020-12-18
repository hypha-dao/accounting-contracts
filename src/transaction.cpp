#include "transaction.hpp"

#include <map>

namespace hypha {

using eosio::check;

Transaction::Transaction(ContentGroups& trxInfo)
{
  ContentWrapper trx(trxInfo);

  auto [hIdx, headerGroup] = trx.getGroup("header");

  check(headerGroup, "Missing 'header' group in transaction document");

  //Check all the header fields are present
  m_memo = trx.getOrFail(hIdx, TRX_MEMO).second->getAs<string>();
  m_date = trx.getOrFail(hIdx, TRX_DATE).second->getAs<time_point>();
  m_ledger = trx.getOrFail(hIdx, TRX_LEDGER).second->getAs<checksum256>();

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
      auto [idx, c] = trx.getOrFail(i, COMPONENT_MEMO, "Missing memo content on trx component");
      component.memo = c->getAs<string>();
    }
    {
      auto [idx, c] = trx.getOrFail(i, COMPONENT_ACCOUNT, "Missing account content on trx component");
      component.account = c->getAs<checksum256>();
    }

    //If there is no ammount, then it should be implied
    if (auto [idx, c] = trx.get(i, COMPONENT_AMMOUNT); c) {
        component.amount = c->getAs<asset>();
        check(component.amount.is_valid(), "Not valid asset: " + 
                                component.amount.to_string() + " at " + 
                                std::to_string(i) + 
                                " memo:" +component.memo);
    }
    

    m_components.emplace_back(std::move(component));
  }  
}

static int64_t 
getSign(int64_t v) 
{
  return v ? static_cast<int64_t>(v > 0) - static_cast<int64_t>(v < 0) : 0;
}

void
Transaction::verifyBalanced()
{
  std::map<uint64_t, asset> assetsBySymb;
  
  for (const auto& component : m_components) {
    auto& asset = component.amount;
    
    auto [assetIt, inserted] = assetsBySymb.insert({asset.symbol.raw(), asset});  

    auto& accum = assetIt->second;

    //It means the asset existed already
    if (!inserted) { accum.set_amount(accum.amount + asset.amount); }
  }

  std::vector<asset> nonZeroAssets;
  //Only 1 asset is allowed to be implied
  asset* impliedAsset = nullptr;

  for (auto& [sym, asset] : assetsBySymb) {
    
    if (asset.symbol) {
      if (asset.amount != 0) { nonZeroAssets.push_back(asset); }
    }
    else { //Implied asset
      check(!impliedAsset, "Only one component is allowed to be implied: " + asset.to_string());
      impliedAsset = &asset;
    }
  }

  if (impliedAsset) {
    //We assume the pending nonZeroAssets 
    //are going to be implied from this asset
    check(!nonZeroAssets.empty(), "Implied asset without remaining compnents");    
  }
  else {
    //Balanced
    if (nonZeroAssets.empty()) { return; }

    //Not valid, there should be at least 2
    check(nonZeroAssets.size() != 1, "Couldn't balance with remaining posting: " + 
                                     nonZeroAssets.back().to_string());
    
    //If there are 2 components left they
    //must cancel each other
    if (nonZeroAssets.size() == 2) {
      check(getSign(nonZeroAssets.front().amount) != getSign(nonZeroAssets.back().amount),
            "Remaining assets must cancel each other" + 
            nonZeroAssets.front().to_string() + ", " + nonZeroAssets.back().to_string());
    }
    else {
      check(false, "Can't balance transaction with remaining components");
    }
  }
}

}