#include "stdafx.h"
#include "VariantHelpers.h"

/// <summary>
/// Function for handling Octet Strings returned by variants.
/// It allocates memory for data using <see cref="HeapAlloc"/>, copies the data to the buffer, and returns a pointer to the buffer.
/// Caller must free the buffer with <see cref="HeapFree"/>.
/// </summary>
/// <param name="variant">Pointer to variant containing the octetstring</param>
/// <param name="outBytes">Return LPBYTE to the data represented in octetstring</param>
/// <returns>The error code.</returns>
HRESULT GetLPBYTEtoOctetString(
	VARIANT* variant,
	LPBYTE* outBytes
)
{
	HRESULT hr = E_FAIL;
	//Check args
	if (!variant || !outBytes)
	{
		return E_INVALIDARG;
	}
	//Check the variant type for unsigned char array (octet string).
	if (variant->vt != (VT_UI1 | VT_ARRAY))
	{
		return E_INVALIDARG;
	}

	void HUGEP* arrayPointer;
	long lLBound, lUBound;
	hr = SafeArrayGetLBound(V_ARRAY(variant), 1, &lLBound);
	hr = SafeArrayGetUBound(V_ARRAY(variant), 1, &lUBound);
	//Get the count of elements
    const auto numElements = lUBound - lLBound + 1;
	hr = SafeArrayAccessData(V_ARRAY(variant), &arrayPointer);
	if (SUCCEEDED(hr))
	{
        const auto tempArray = static_cast<LPBYTE>(arrayPointer);
		*outBytes = static_cast<LPBYTE>(HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, numElements));
		if (*outBytes)
		{
			memcpy(*outBytes, tempArray, numElements);
		}
		else
		{
			hr = E_OUTOFMEMORY;
		}
		SafeArrayUnaccessData(V_ARRAY(variant));
	}
	return hr;
}

std::tuple<sid_ptr, HRESULT> VariantToSID(VARIANT* variant)
{
	PSID psid = nullptr;
	HRESULT hr = GetLPBYTEtoOctetString(variant, reinterpret_cast<LPBYTE*>(&psid));
	if (FAILED(hr))
    {
		return std::make_tuple(nullptr, hr);
    }
	sid_ptr sid(reinterpret_cast<sid_ptr::pointer>(psid));
	return std::make_tuple(std::move(sid), hr);
}
