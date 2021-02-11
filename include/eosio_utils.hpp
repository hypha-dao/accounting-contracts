#pragma once

#include <eosio/crypto.hpp>

namespace hypha {
namespace util {

inline eosio::checksum256 
hashString(const std::string& str) 
{
  return eosio::sha256(str.data(), str.size());
}

}
}