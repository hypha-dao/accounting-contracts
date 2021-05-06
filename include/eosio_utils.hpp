#pragma once

#include <cmath>
#include <string>

#include <eosio/eosio.hpp>
#include <eosio/asset.hpp>

#include <eosio/crypto.hpp>
#include <document_graph/util.hpp>

namespace hypha {
namespace util {

inline eosio::checksum256 
hashString(const std::string& str) 
{
  return eosio::sha256(str.data(), str.size());
}

inline float
calculateRate(eosio::asset from, eosio::asset to)
{
  double fromTotal = static_cast<double>(from.amount)  / 
                     static_cast<double>(std::powf(10, from.symbol.precision()));

  double toTotal = static_cast<double>(to.amount) /
                   static_cast<double>(std::powf(10, from.symbol.precision()));

  return static_cast<float>(fromTotal/toTotal);
}

}
}