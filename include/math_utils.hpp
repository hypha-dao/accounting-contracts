#pragma once

#include <eosio/eosio.hpp>
#include <eosio/asset.hpp>

namespace hypha {
  namespace util {

    static int64_t 
    integerPow(int64_t x, int64_t p)
    {
      if (p == 0) return 1;
      if (p == 1) return x;
      
      int64_t tmp = integerPow(x, p/2);
      if (p%2 == 0) return tmp * tmp;
      else return x * tmp * tmp;
    }

    static eosio::asset
    addAssetsAdjustingPrecision(const eosio::asset a, const eosio::asset b)
    {
      eosio::asset res1, res2;

      if (a.symbol.precision() < b.symbol.precision()) {
        res1 = b;
        res2 = eosio::asset(a.amount * integerPow(10, b.symbol.precision() - a.symbol.precision()), b.symbol);
      } else {
        res1 = a;
        res2 = eosio::asset(b.amount * integerPow(10, a.symbol.precision() - b.symbol.precision()), a.symbol);
      }
      
      return res1 + res2;
    }

    static double
    asset2double(const eosio::asset & quantity)
    {
      double amount = quantity.amount;
      return amount / double(integerPow(10, quantity.symbol.precision()));
    }

  }
}