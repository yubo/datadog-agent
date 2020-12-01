#pragma once

#include <tuple>
#include "SID.h"

std::tuple<sid_ptr, HRESULT> SearchActiveDirectory(std::wstring const& username, std::wstring const& context);
std::wstring DomainNameToLDAPDataInterchangeFormat(std::wstring const& domainName);
