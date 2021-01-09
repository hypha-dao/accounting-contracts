#pragma once

#include <string_view>
#include <optional>

#include <document_graph/document.hpp>

#include "constants.hpp"

namespace hypha
{

using eosio::check;

//Singleton settings object
//Settings added by the setsetting action will be stored
//in the SETTINGS_DATA group
class Settings
{
 public:
  
  static Settings& 
  instance();

  template<class T>
  inline std::optional<T>
  getOpt(const string& setting, const char* group = SETTINGS_DATA)
  {
    ContentWrapper cw(m_settings.getContentGroups());
    
    if (auto [idx, content] = cw.get(group, setting); content) {
      if (auto value = std::get_if<T>(content->value)) {
        return std::optional{*value};
      }
    }

    return {};
  }

  template<class T>
  inline T
  getOrFail(const string& setting, const char* group = SETTINGS_DATA) 
  {
    if (auto opt = getOpt<T>(setting, group)) {
      return *opt;
    }

    check(false, "Setting: " + setting + " required but not found");
    //Just to avoid warnings
    return T{};
  }

  template<class T>
  inline T
  getOrDefault(const string& setting, const T& defVal = T(), const char* group = SETTINGS_DATA)
  {
    if (auto opt = getOpt<T>(setting, group)) {
      return *opt;
    }

    return defVal;
  }  

  inline ContentWrapper
  getWrapper() 
  {
    return ContentWrapper(m_settings.getContentGroups());
  }

  void 
  addOrReplace(const string& setting, 
               Content::FlexValue value, 
               const char* groupName = SETTINGS_DATA);

  void
  remove(const string& setting, const char* groupName = SETTINGS_DATA);

  void
  remove(const Content& setting, const char* groupName = SETTINGS_DATA);

 private:

  Settings();

  Document m_settings;
};

} // namespace hypha

