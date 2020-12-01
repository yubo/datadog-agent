#include "stdafx.h"

#include <ActiveDS.h>
#include <objbase.h>
#include <comutil.h>
#include <sddl.h>
#include <comip.h>

#include <sstream>

#include "ActiveDirectory.h"
#include "VariantHelpers.h"

typedef _com_ptr_t <_com_IIID<IADs, &__uuidof(IADs)>> IADsPtr;
typedef _com_ptr_t <_com_IIID<IDirectorySearch, &__uuidof(IDirectorySearch)>> IDirectorySearchPtr;

std::tuple<sid_ptr, HRESULT> SearchActiveDirectory(std::wstring const& username, std::wstring const& context)
{
	WcaLog(LOGMSG_STANDARD, "Searching LDAP for %S", username.c_str());
	IADsPtr activeDirectory;
	HRESULT hr = ADsOpenObject(L"LDAP://rootDSE",
		nullptr,
		nullptr,
		ADS_SECURE_AUTHENTICATION,
		IID_IADs,
		reinterpret_cast<void**>(&activeDirectory));
	if (FAILED(hr))
	{
		WcaLog(LOGMSG_STANDARD, "Cannot execute query. Cannot bind to LDAP://rootDSE.");
		return std::make_tuple(nullptr, hr);
	}

	VARIANT var;
	std::wstring adPath = L"LDAP://";
	if (context.empty())
	{
		hr = activeDirectory->Get(_bstr_t(L"defaultNamingContext"), &var);
		if (SUCCEEDED(hr))
		{
			//  Build path to the domain container.
			adPath += var.bstrVal;
		}
		else
		{
			WcaLog(LOGMSG_STANDARD, "Failed to get default naming context.");
			return std::make_tuple(nullptr, hr);
		}
	}
	else
	{
		adPath += context;
	}

	WcaLog(LOGMSG_STANDARD, "Looking into %S", adPath.c_str());
	IDirectorySearchPtr directorySearcher;
	hr = ADsOpenObject(adPath.c_str(),
		nullptr,
		nullptr,
		ADS_SECURE_AUTHENTICATION,
		IID_IDirectorySearch,
		reinterpret_cast<void**>(&directorySearcher));
	if (FAILED(hr))
	{
		WcaLog(LOGMSG_STANDARD, "Cannot execute query. Cannot open defaultNamingContext");
		return std::make_tuple(nullptr, hr);
	}

	ADS_SEARCHPREF_INFO searchPrefs;
	searchPrefs.dwSearchPref = ADS_SEARCHPREF_SEARCH_SCOPE;
	searchPrefs.vValue.dwType = ADSTYPE_INTEGER;
	searchPrefs.vValue.Integer = ADS_SCOPE_SUBTREE;
	DWORD numPrefs = 1;
	hr = directorySearcher->SetSearchPreference(&searchPrefs, numPrefs);
	if (FAILED(hr))
	{
		WcaLog(LOGMSG_STANDARD, "Cannot execute query. SetSearchPreference failed.");
		return std::make_tuple(nullptr, hr);
	}

	std::wstringstream filter;
	filter << L"(&(objectClass=user)(objectCategory=person)(cn=" << username << L"))";
	LPOLESTR attributes[] = { L"ADsPath" };
	ADS_SEARCH_HANDLE searchHandle = nullptr;

	// ExecuteSearch doesn't modify the filter, it's fine to const_cast
	hr = directorySearcher->ExecuteSearch(const_cast<wchar_t*>(filter.str().c_str()),
		attributes,
		sizeof(attributes) / sizeof(LPOLESTR),
		&searchHandle);

	std::tuple<sid_ptr, HRESULT> searchResult = std::make_tuple(nullptr, E_FAIL);
	if (SUCCEEDED(hr))
	{
		while (directorySearcher->GetNextRow(searchHandle) != S_ADS_NOMORE_ROWS)
		{
			LPOLESTR colunmName = nullptr;
			ADS_SEARCH_COLUMN col;
			while (directorySearcher->GetNextColumnName(searchHandle, &colunmName) != S_ADS_NOMORE_COLUMNS)
			{
				hr = directorySearcher->GetColumn(searchHandle, attributes[0], &col);
				if (SUCCEEDED(hr))
				{
					std::wstring adpath = col.pADsValues->CaseIgnoreString;
					IADsPtr adUser;
					hr = ADsOpenObject(adpath.c_str(),
						nullptr,
						nullptr,
						ADS_SECURE_AUTHENTICATION, //Use Secure Authentication
						IID_IADs,
						reinterpret_cast<void**>(&adUser));
					if (SUCCEEDED(hr))
					{
						hr = adUser->Get(_bstr_t(L"objectSid"), &var);
						// There should only be one column
                        // but we should let the rest of the loop
                        // execute to cleanup
						if (FAILED(hr))
						{
							WcaLog(LOGMSG_STANDARD, "Get PSID failed with hr: %x");
						}
						else
						{
							searchResult = VariantToSID(&var);
						}
					}
					directorySearcher->FreeColumn(&col);
				}
			}
			FreeADsMem(colunmName);
			hr = directorySearcher->GetNextRow(searchHandle);
		}
	}
	directorySearcher->CloseSearchHandle(searchHandle);

	return searchResult;
}

std::wstring DomainNameToLDAPDataInterchangeFormat(std::wstring const& domainName)
{
	std::wstringstream domainLdif;
	auto dotOffset = domainName.find('.');
	auto previousOffset = 0;
	while (dotOffset != std::wstring::npos)
	{
		if (previousOffset > 0)
		{
			domainLdif << ",";
		}
		domainLdif << "DC=" << domainName.substr(previousOffset, dotOffset - previousOffset);
		previousOffset = dotOffset + 1;
		dotOffset = domainName.find('.', previousOffset);
		if (dotOffset == std::wstring::npos)
		{
			domainLdif << ",DC=" << domainName.substr(previousOffset);
		}
	}
	return domainLdif.str();
}
