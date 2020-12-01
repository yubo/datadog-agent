#pragma once

#include <memory>
#include <string>
#include <winnt.h>
#include "CustomDeleters.h"

/// <summary>
/// Defines a smart pointer for <see cref="SID">.
/// </summary>
typedef std::unique_ptr<SID, heap_deleter<SID>> sid_ptr;

/// <summary>
/// Creates a new <see cref="SID"> from a length.
/// Used in Win32 API that write arbitrary length buffers to be interpreted as an <see cref="SID">.
/// </summary>
/// <param name="sidLength">The length of the SID to allocate.</param>
/// <returns>A <see cref="sid_ptr"/> managing a buffer allocated from the heap.</returns>
sid_ptr make_sid(size_t sidLength);

/// <summary>
/// Retrieves the Security Identifier Descriptor of the specified user.
/// </summary>
/// <param name="host">The host to search on.</param>
/// <param name="user">The username to look for.</param>
/// <returns>A tuple containing a pointer to the SID of the user and and error code.
/// If no user is found, the pointer to the SID will be NULL and the DWORD will contain  the result of <see cref="GetLastError">.</returns>
std::tuple<sid_ptr, HRESULT> GetSidForUser(LPCWSTR host, LPCWSTR user);

bool GetNameForSid(LPCWSTR host, PSID sid, std::wstring& namestr);
