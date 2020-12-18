#pragma once

#include <string_view>
#include <optional>

#include <document_graph/document.hpp>

#include "constants.hpp"

namespace hypha
{

using eosio::check;

//Singleton settings object
class Settings
{
 public:
  
  static Settings& 
  instance();

  template<class T>
  inline std::optional<T>
  getOpt(const string& setting)
  {
    ContentWrapper cw(m_settings.getContentGroups());
    
    if (auto [idx, content] = cw.get(SETTINGS_DATA, setting); content) {
      if (auto value = std::get_if<T>(content->value)) {
        return std::optional{*value};
      }
    }

    return {};
  }

  template<class T>
  inline T
  getOrFail(const string& setting) 
  {
    if (auto opt = getOpt<T>(setting)) {
      return *opt;
    }

    check(false, "Setting: " + setting + " required but not found");
    //Just to avoid warnings
    return T{};
  }

  template<class T>
  inline T
  getOrDefault(const string& setting, const T& defVal = T())
  {
    if (auto opt = getOpt<T>(setting)) {
      return *opt;
    }

    return defVal;
  }

  void 
  addOrReplace(const string& setting, Content::FlexValue value);

  void
  remove(const string& setting);

 private:

  Settings();

  Document m_settings;
};

} // namespace hypha

