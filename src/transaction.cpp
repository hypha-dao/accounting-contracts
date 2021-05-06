#include "transaction.hpp"

#include <map>

#include <logger/logger.hpp>

namespace hypha {

using eosio::check;

Transaction::Transaction(ContentGroups& trxInfo)
{
  TRACE_FUNCTION()

  ContentWrapper trx(trxInfo);

  auto [hIdx, headerGroup] = trx.getGroup("header");

  EOS_CHECK(headerGroup, "Missing 'header' group in transaction document");

  //Check all the header fields are present
  m_memo = trx.getOrFail(hIdx, TRX_MEMO).second->getAs<string>();
  m_date = trx.getOrFail(hIdx, TRX_DATE).second->getAs<time_point>();
  m_ledger = trx.getOrFail(hIdx, TRX_LEDGER).second->getAs<checksum256>();

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

    //If there is no ammount, then it should be implied
    if (auto [idx, c] = trx.get(i, COMPONENT_AMMOUNT); c) {
        component.amount = c->getAs<asset>();
        EOS_CHECK(component.amount.is_valid(), "Not valid asset: " + 
                                component.amount.to_string() + " at " + 
                                std::to_string(i) + 
                                " memo:" +component.memo);
    }
    
    LOG_MESSAGE(util::to_str("Component read: ", trxInfo[i]))

    m_components.emplace_back(std::move(component));
  }
}

static int64_t 
getSign(int64_t v) 
{
  return v ? static_cast<int64_t>(v > 0) - static_cast<int64_t>(v < 0) : 0;
}

std::vector<asset>
Transaction::verifyBalanced()
{
  TRACE_FUNCTION()

  std::map<uint64_t, asset> assetsBySymb;
  std::vector<asset> allAssets;
  
  for (const auto& component : m_components) {
    auto& asset = component.amount;
    
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
      EOS_CHECK(!impliedAsset, "Only one component is allowed to be implied: " + asset.to_string());
      impliedAsset = &asset;
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

}