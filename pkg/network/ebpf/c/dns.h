#ifndef __DNS_H
#define __DNS_H

// This function is meant to be used as a BPF_PROG_TYPE_SOCKET_FILTER.
// When attached to a RAW_SOCKET, this code filters out everything but DNS traffic.
// All structs referenced here are kernel independent as they simply map protocol headers (Ethernet, IP and UDP).
SEC("socket/dns_filter")
int socket__dns_filter(struct __sk_buff* skb) {
    skb_info_t skb_info;

    if (!read_conn_tuple_skb(skb, &skb_info)) {
        return 0;
    }

    if (skb_info.tup.sport != 53 && (!dns_stats_enabled() || skb_info.tup.dport != 53)) {
        return 0;
    }

    return -1;
}

#endif
