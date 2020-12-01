#pragma once

#include "SID.h"

std::tuple<sid_ptr, HRESULT> VariantToSID(VARIANT* variant);
