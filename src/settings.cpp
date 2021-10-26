#include "settings.hpp"

#include "accounting.hpp"

namespace hypha {

static ContentGroups
getInitalGroups() 
{
  return {
    ContentGroup {
      Content{CONTENT_GROUP_LABEL, DETAILS},
      Content{ROOT_NODE, accounting::getRoot().getHash()},
      // Content{"name", SETTINGS}
    },
    ContentGroup {
      Content{CONTENT_GROUP_LABEL, SETTINGS_DATA}
    },
    accounting::getSystemGroup("settings", "settings")
  };
}

Settings::Settings()
{
  auto root = accounting::getRoot();

  if (auto [exists, edge] = Edge::getIfExists(accounting::getName(), 
                                              accounting::getRoot().getHash(),
                                              SETTINGS_EDGE); exists) {
    m_settings = Document(accounting::getName(), edge.to_node);
  }
  else {
    m_settings = Document(accounting::getName(), accounting::getName(), getInitalGroups());
    Edge(accounting::getName(), 
         accounting::getName(), 
         root.getHash(), m_settings.getHash(), SETTINGS_EDGE);
  }
}

void 
Settings::addOrReplace(const string& setting, Content::FlexValue value, const char* groupName)
{
  ContentWrapper cw(m_settings.getContentGroups());
  
  auto group = cw.getGroupOrFail(groupName);

  ContentWrapper::insertOrReplace(*group, Content{setting, value});

  ContentWrapper::insertOrReplace(*group, Content{UPDATE_DATE, eosio::current_time_point()});

  auto dgraph = DocumentGraph(accounting::getName());

  m_settings = dgraph.updateDocument(accounting::getName(), 
                                     m_settings.getHash(),
                                     m_settings.getContentGroups());
}

void 
Settings::add(const string& setting, 
              Content::FlexValue value, 
              const char* groupName) 
{
  ContentWrapper cw(m_settings.getContentGroups());

  auto group = cw.getGroupOrFail(groupName);

  group->push_back(Content{setting, value});

  auto dgraph = DocumentGraph(accounting::getName());

  m_settings = dgraph.updateDocument(accounting::getName(), 
                                     m_settings.getHash(),
                                     m_settings.getContentGroups());
}

void
Settings::remove(const string& setting, const char* groupName)
{
  ContentWrapper cw(m_settings.getContentGroups());

  cw.removeContent(groupName, setting);

  auto dgraph = DocumentGraph(accounting::getName());

  m_settings = dgraph.updateDocument(accounting::getName(), 
                                     m_settings.getHash(),
                                     m_settings.getContentGroups());
}

void Settings::remove(const Content& setting, const char* groupName) 
{
  ContentWrapper cw(m_settings.getContentGroups());

  cw.removeContent(groupName, setting);

  auto dgraph = DocumentGraph(accounting::getName());

  m_settings = dgraph.updateDocument(accounting::getName(), 
                                     m_settings.getHash(),
                                     m_settings.getContentGroups());
}

void Settings::save()
{
  auto dgraph = DocumentGraph(accounting::getName());
  m_settings = dgraph.updateDocument(accounting::getName(), 
                                     m_settings.getHash(),
                                     m_settings.getContentGroups());  
}

Settings& 
Settings::instance() 
{
  static Settings s;
  return s;
}

}