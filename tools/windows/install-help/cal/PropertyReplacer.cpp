#include "stdafx.h"
#include "PropertyReplacer.h"

namespace
{
    template <class Map>
    bool has_key(Map const &m, const typename Map::key_type &key)
    {
        auto const &it = m.find(key);
        return it != m.end();
    }

    std::wstring format_tags(std::map<std::wstring, std::wstring> &values)
    {
        std::wistringstream valueStream(values[L"TAGS"]);
        std::wstringstream result;
        std::wstring token;
        result << L"tags: ";
        while (std::getline(valueStream, token, wchar_t(',')))
        {
            result << std::endl << L"  - " << token;
        }
        return result.str();
    };

    std::wstring format_proxy(std::map<std::wstring, std::wstring> &values)
    {
        const auto &proxyHost = values.find(L"PROXY_HOST");
        const auto &proxyPort = values.find(L"PROXY_PORT");
        const auto &proxyUser = values.find(L"PROXY_USER");
        const auto &proxyPassword = values.find(L"PROXY_PASSWORD");
        std::wstringstream proxy;
        if (proxyUser != values.end())
        {
            proxy << proxyUser->second;
            if (proxyPassword != values.end())
            {
                proxy << L":" << proxyPassword->second;
            }
            proxy << L"@";
        }
        proxy << proxyHost->second;
        if (proxyPort != values.end())
        {
            proxy << L":" << proxyPort->second;
        }
        std::wstringstream newValue;
        newValue << L"proxy:" << std::endl
                 << L"  https: " << proxy.str() << std::endl
                 << L"  http: " << proxy.str() << std::endl;
        return newValue.str();
    };

} // namespace

PropertyReplacer::PropertyReplacer(std::wstring &input, std::wstring const &match)
    : _input(input)
{
    _matches.push_back(std::wregex(match));
}

bool PropertyReplacer::replace_with(std::wstring const &replacement)
{
    auto start = _input.begin();
    auto end = _input.end();
    std::size_t offset = 0;
    for (auto matchIt = _matches.begin(); matchIt != _matches.end();)
    {
        std::match_results<decltype(start)> results;
        if (!std::regex_search(start + offset, end, results, *matchIt, std::regex_constants::format_first_only))
        {
            return false;
        }
        if (++matchIt == _matches.end())
        {
            _input.erase(offset + results.position(), results.length());
            _input.insert(offset + results.position(), replacement);
        }
        else
        {
            offset += results.position();
        }
    }
    return true;
}

PropertyReplacer &PropertyReplacer::then(std::wstring const &match)
{
    _matches.push_back(std::wregex(match));
    return *this;
}

PropertyReplacer PropertyReplacer::match(std::wstring &input, std::wstring const &match)
{
    return PropertyReplacer(input, match);
}

std::wstring replace_yaml_properties(std::wstring input, value_map &values)
{
    enum PropId
    {
        WxsKey,
        Regex,
        Replacement
    };
    typedef std::function<std::wstring(value_map &)> formatter_func;
    typedef std::vector<std::tuple<std::wstring, std::wstring, formatter_func>> prop_list;
    for (auto prop : prop_list{
        {L"APIKEY",       L"^[ #]*api_key:.*",        [](auto &v) { return L"api_key: " + v[L"APIKEY"]; }},
        {L"SITE",         L"^[ #]*site:.*",           [](auto &v) { return L"site: " + v[L"SITE"]; }},
        {L"HOSTNAME",     L"^[ #]*hostname:.*",       [](auto &v) { return L"hostname: " + v[L"HOSTNAME"]; }},
        {L"LOGS_ENABLED", L"^[ #]*logs_enabled:.*",   [](auto &v) { return L"logs_enabled: " + v[L"LOGS_ENABLED"]; }},
        {L"CMD_PORT",     L"^[ #]*cmd_port:.*",       [](auto &v) { return L"cmd_port: " + v[L"CMD_PORT"]; }},
        {L"DD_URL",       L"^[ #]*dd_url:.*",         [](auto &v) { return L"dd_url: " + v[L"DD_URL"]; }},
        {L"PYVER",        L"^[ #]*python_version:.*", [](auto &v) { return L"python_version:" + v[L"PYVER"]; }},
        // This replacer will uncomment the logs_config section if LOGS_DD_URL is specified, regardless of its value
        {L"LOGS_DD_URL",  L"^[ #]*logs_config:.*",    [](auto &v) { return L"logs_config:"; }},
        // logs_dd_url and apm_dd_url are indented so override default formatter to specify correct indentation
        {L"LOGS_DD_URL",  L"^[ #]*logs_dd_url:.*",    [](auto &v) { return L"  logs_dd_url:" + v[L"LOGS_DD_URL"]; }},
        {L"TRACE_DD_URL", L"^[ #]*apm_dd_url:.*",     [](auto &v) { return L"  apm_dd_url:" + v[L"TRACE_DD_URL"]; }},
        {L"TAGS",         L"^[ #]*tags:(?:(?:.|\n)*?)^[ #]*- <TAG_KEY>:<TAG_VALUE>", format_tags},
        {L"PROXY_HOST",   L"^[ #]*proxy:.*", format_proxy},
        {L"HOSTNAME_FQDN_ENABLED", L"^[ #]*hostname_fqdn:.*", [](auto &v) { return L"hostname_fqdn:" + v[L"hostname_fqdn"]; }},
    })
    {
        if (has_key(values, std::get<WxsKey>(prop)))
        {
            PropertyReplacer::match(input, std::get<Regex>(prop)).replace_with(std::get<Replacement>(prop)(values));
        }
    }

    // Special cases
    if (has_key(values, L"PROCESS_ENABLED"))
    {

        if (has_key(values, L"PROCESS_DD_URL"))
        {
            PropertyReplacer::match(input, L"^[ #]*process_config:")
                .replace_with(L"process_config:\n  process_dd_url: " + values[L"PROCESS_DD_URL"]);
        }
        else
        {
            PropertyReplacer::match(input, L"^[ #]*process_config:").replace_with(L"process_config:");
        }

        PropertyReplacer::match(input, L"process_config:")
            .then(L"^[ #]*enabled:.*")
            // Note that this is a string, and should be between ""
            .replace_with(L"  enabled: \"" + values[L"PROCESS_ENABLED"] + L"\"");
    }

    if (has_key(values, L"APM_ENABLED"))
    {
        PropertyReplacer::match(input, L"^[ #]*apm_config:").replace_with(L"apm_config:");
        PropertyReplacer::match(input, L"apm_config:")
            .then(L"^[ #]*enabled:.*")
            .replace_with(L"  enabled: " + values[L"APM_ENABLED"]);
    }

    return input;
}
