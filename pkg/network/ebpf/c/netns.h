#ifndef __NETNS_H
#define __NETNS_H

#include "defs.h"

static __always_inline __u64 offset_netns() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_netns", val);
    return val;
}

static __always_inline __u64 offset_ino() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_ino", val);
    return val;
}

static __always_inline __u32 get_netns_from_sock(struct sock* sk) {
    possible_net_t* skc_net = NULL;
    __u32 net_ns_inum = 0;
    bpf_probe_read(&skc_net, sizeof(possible_net_t*), ((char*)sk) + offset_netns());
    bpf_probe_read(&net_ns_inum, sizeof(net_ns_inum), ((char*)skc_net) + offset_ino());
    return net_ns_inum;
}

#endif
