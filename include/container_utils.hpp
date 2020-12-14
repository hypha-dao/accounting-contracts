#include <string>
#include <optional>
#include <string_view>

namespace hypha {
namespace util {

using std::string;
using std::string_view;

inline string 
toLowerCase(string original)
{
  for (auto& c : original) {
    c = ::tolower(c);
  }

  return original;
}

inline string
toUpperCase(string original) {
  for (auto& c : original) {
    c = ::toupper(c);
  }

  return original;
}

inline bool
containsPrefix(string_view str, string_view prefix)
{
  if (prefix.size() > str.size()) { return false; }

  auto start = str.begin();

  for (auto c : prefix) {
    if (c != *(start++)) { return false; }
  }

  return true;
}

/**
* @brief Gets the rest of the string after the last appearence of
* the specified character
*/ 
inline string_view
getSubstrAfterLastOf(string_view str, char search) {
  if (auto lastPos = str.find_last_of(search); 
      lastPos != str.npos) {
    return {str.begin()+lastPos+1};
  }

  return {};
}

}
}