#pragma once
#include <functional>
#include <regex>
#include <string>

class PropertyReplacer
{
private:
    std::wstring &_input;
    std::vector<std::wregex> _matches;
    PropertyReplacer(std::wstring &input, std::wstring const &match);
  public:
    bool replace_with(std::wstring const &replacement);

    PropertyReplacer &then(std::wstring const &match);

    static PropertyReplacer match(std::wstring &input, std::wstring const &match);
};

typedef std::map<std::wstring, std::wstring> value_map;
std::wstring replace_yaml_properties(std::wstring input, value_map &values);
