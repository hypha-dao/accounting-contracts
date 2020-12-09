#include <eosio/eosio.hpp>

namespace hypha {

using eosio::name;

template<class T>
class TableWrapper
{

 public:

  TableWrapper(name code, uint64_t scope) : m_table(code, scope) 
  {}

  template<name::raw Index, class U>
  inline auto
  find_by(const U& at)
  {
    return index_by<Index>().find(at);
  }

  template<name::raw Index, class U>
  inline auto
  contains_by(const U& at)
  {
    return find_by<Index>(at) != end_by<Index>();
  }

  inline auto
  find(uint64_t index) const
  {
    return m_table.find(index);
  }

  inline bool
  contains(uint64_t index)
  {
    return find(index) != end();
  }
  
  template<name::raw Index, class U>
  inline decltype(auto)
  get_by(const U& at, const char* error = "unable to find key") const
  {
    return index_by<Index>().get(at, error);
  }

  inline decltype(auto)
  get(uint64_t index, const char* error = "unable to find key") const
  {
    return m_table.get(index, error);
  }

  template<name::raw Index>
  inline auto
  index_by() const
  {
    return m_table.template get_index<Index>();
  }

  template<name::raw Index>
  inline auto
  end_by() const
  {
    return index_by<Index>().end();
  }

  template<name::raw Index>
  inline auto
  begin_by() const
  {
    return index_by<Index>().begin();
  }

  inline auto
  end() const
  {
    return m_table.end();
  }

  inline auto
  begin() const
  {
    return m_table.begin();
  }

  using StoreType = typename T::const_iterator::value_type;

  template<class Emplacer>
  inline auto
  insert(name payer, const Emplacer& emplacer)
  {
    m_table.emplace(payer, emplacer);
  }
 private:
  T m_table; 
};

}