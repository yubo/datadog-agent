#pragma once

/// <summary>
/// Defines a custom deleter that frees memory
/// allocated from the progress heap using <see cref="HeapAlloc"/>
/// </summary>
/// <typeparam name="P">The type of the pointer to free.</typeparam>
template <class P>
struct heap_deleter
{
    typedef P* pointer;

    void operator()(pointer ptr) const
    {
        HeapFree(GetProcessHeap(), 0, ptr);
    }
};


template <class P>
struct release_deleter
{
    typedef P* pointer;

    void operator()(pointer ptr) const
    {
        ptr->Release();
    }
};
