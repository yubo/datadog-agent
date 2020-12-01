#include "stdafx.h"
#include "SID.h"

sid_ptr make_sid(size_t sidLength)
{
    return sid_ptr(static_cast<sid_ptr::pointer>(HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, sidLength)));
}

std::tuple<sid_ptr, HRESULT> GetSidForUser(LPCWSTR host, LPCWSTR user)
{
    DWORD cbSid = 0;
    DWORD cchRefDomain = 0;
    SID_NAME_USE use;

    LookupAccountName(host, user, nullptr, &cbSid, nullptr, &cchRefDomain, &use);
    sid_ptr newsid = make_sid(cbSid);
    std::unique_ptr<wchar_t[]> refDomain(new wchar_t[cchRefDomain + 1]);
    if (!LookupAccountName(host, user, newsid.get(), &cbSid, refDomain.get(), &cchRefDomain, &use))
    {
        return std::make_tuple(nullptr, GetLastError());
    }
    WcaLog(LOGMSG_VERBOSE, "Got SID from %S", refDomain.get());
    if (!IsValidSid(newsid.get()))
    {
        WcaLog(LOGMSG_STANDARD, "New SID is invalid");
        return std::make_tuple(nullptr, ERROR_INVALID_SID);
    }

    return std::make_tuple(std::move(newsid), ERROR_SUCCESS);
}

bool GetNameForSid(LPCWSTR host, PSID sid, std::wstring& namestr)
{
    wchar_t* name = NULL;
    DWORD cchName = 0;
    LPWSTR refDomain = NULL;
    DWORD cchRefDomain = 0;
    SID_NAME_USE use;
    BOOL success = false;
    BOOL bRet = LookupAccountSid(host, sid, name, &cchName, refDomain, &cchRefDomain, &use);
    if (bRet) {
        // this should *never* happen, because we didn't pass in a buffer large enough for
        // the sid or the domain name.
        WcaLog(LOGMSG_STANDARD, "Unexpected success looking up account sid");
        return false;
    }
    DWORD err = GetLastError();
    if (ERROR_INSUFFICIENT_BUFFER != err) {
        WcaLog(LOGMSG_STANDARD, "Unexpected failure looking up account sid %d", err);
        // we don't know what happened
        return false;
    }
    name = (wchar_t*) new wchar_t[cchName];
    ZeroMemory(name, cchName * sizeof(wchar_t));

    refDomain = new wchar_t[cchRefDomain + 1];
    ZeroMemory(refDomain, (cchRefDomain + 1) * sizeof(wchar_t));

    // try it again
    bRet = LookupAccountSid(host, sid, name, &cchName, refDomain, &cchRefDomain, &use);
    if (!bRet) {
        WcaLog(LOGMSG_STANDARD, "Failed to lookup account name %d", GetLastError());
        goto cleanAndDone;
    }
    success = true;
    WcaLog(LOGMSG_STANDARD, "Got account sid from %S\n", refDomain);
    namestr = name;

cleanAndDone:
    if (name) {
        delete[](wchar_t*)name;
    }
    if (refDomain) {
        delete[] refDomain;
    }
    return success;
}
