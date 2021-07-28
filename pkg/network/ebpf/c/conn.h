#ifndef __CONN_H
#define __CONN_H

#include "defs.h"

static __always_inline __u64 offset_family() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_family", val);
    return val;
}

static __always_inline __u64 offset_saddr() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_saddr", val);
    return val;
}

static __always_inline __u64 offset_daddr() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_daddr", val);
    return val;
}

static __always_inline __u64 offset_daddr_ipv6() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_daddr_ipv6", val);
    return val;
}

static __always_inline __u64 offset_sport() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_sport", val);
    return val;
}

static __always_inline __u64 offset_dport() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_dport", val);
    return val;
}

static __always_inline bool is_ipv6_enabled() {
    __u64 val = 0;
    LOAD_CONSTANT("ipv6_enabled", val);
    return val == ENABLED;
}

static __always_inline __u16 read_sport(struct sock* sk) {
    __u16 sport = 0;
    // try skc_num, then inet_sport
    bpf_probe_read(&sport, sizeof(sport), ((char*)sk) + offset_dport() + sizeof(sport));
    if (sport == 0) {
        bpf_probe_read(&sport, sizeof(sport), ((char*)sk) + offset_sport());
        sport = bpf_ntohs(sport);
    }
    return sport;
}

static __always_inline bool check_family(struct sock* sk, u16 expected_family) {
    u16 family = 0;
    bpf_probe_read(&family, sizeof(u16), ((char*)sk) + offset_family());
    return family == expected_family;
}


/**
 * Reads values into a `conn_tuple_t` from a `sock`. Any values that are already set in conn_tuple_t
 * are not overwritten. Returns 1 success, 0 otherwise.
 */
static __always_inline int read_conn_tuple_partial(conn_tuple_t * t, struct sock* skp, u64 pid_tgid, metadata_mask_t type) {
    t->pid = pid_tgid >> 32;
    t->metadata = type;

    // Retrieve network namespace id first since addresses and ports may not be available for unconnected UDP
    // sends
    t->netns = get_netns_from_sock(skp);

    // Retrieve addresses
    if (check_family(skp, AF_INET)) {
        t->metadata |= CONN_V4;
        if (t->saddr_l == 0) {
            bpf_probe_read(&t->saddr_l, sizeof(u32), ((char*)skp) + offset_saddr());
        }
        if (t->daddr_l == 0) {
            bpf_probe_read(&t->daddr_l, sizeof(u32), ((char*)skp) + offset_daddr());
        }

        if (!t->saddr_l || !t->daddr_l) {
            log_debug("ERR(read_conn_tuple.v4): src or dst addr not set src=%d, dst=%d\n", t->saddr_l, t->daddr_l);
            return 0;
        }
    } else if (check_family(skp, AF_INET6)) {
        if (!is_ipv6_enabled()) {
            return 0;
        }

        if (t->saddr_h == 0) {
            bpf_probe_read(&t->saddr_h, sizeof(t->saddr_h), ((char*)skp) + offset_daddr_ipv6() + 2 * sizeof(u64));
        }
        if (t->saddr_l == 0) {
            bpf_probe_read(&t->saddr_l, sizeof(t->saddr_l), ((char*)skp) + offset_daddr_ipv6() + 3 * sizeof(u64));
        }
        if (t->daddr_h == 0) {
            bpf_probe_read(&t->daddr_h, sizeof(t->daddr_h), ((char*)skp) + offset_daddr_ipv6());
        }
        if (t->daddr_l == 0) {
            bpf_probe_read(&t->daddr_l, sizeof(t->daddr_l), ((char*)skp) + offset_daddr_ipv6() + sizeof(u64));
        }

        // We can only pass 4 args to bpf_trace_printk
        // so split those 2 statements to be able to log everything
        if (!(t->saddr_h || t->saddr_l)) {
            log_debug("ERR(read_conn_tuple.v6): src addr not set: type=%d, saddr_l=%d, saddr_h=%d\n",
                      type, t->saddr_l, t->saddr_h);
            return 0;
        }

        if (!(t->daddr_h || t->daddr_l)) {
            log_debug("ERR(read_conn_tuple.v6): dst addr not set: type=%d, daddr_l=%d, daddr_h=%d\n",
                      type, t->daddr_l, t->daddr_h);
            return 0;
        }

        // Check if we can map IPv6 to IPv4
        if (is_ipv4_mapped_ipv6(t->saddr_h, t->saddr_l, t->daddr_h, t->daddr_l)) {
            t->metadata |= CONN_V4;
            t->saddr_h = 0;
            t->daddr_h = 0;
            t->saddr_l = (__u32)(t->saddr_l >> 32);
            t->daddr_l = (__u32)(t->daddr_l >> 32);
        } else {
            t->metadata |= CONN_V6;
        }
    }

    // Retrieve ports
    if (t->sport == 0) {
        t->sport = read_sport(skp);
    }
    if (t->dport == 0) {
        bpf_probe_read(&t->dport, sizeof(t->dport), ((char*)skp) + offset_dport());
        t->dport = bpf_ntohs(t->dport);
    }

    if (t->sport == 0 || t->dport == 0) {
        log_debug("ERR(read_conn_tuple.v4): src/dst port not set: src:%d, dst:%d\n", t->sport, t->dport);
        return 0;
    }

    return 1;
}

/**
 * Reads values into a `conn_tuple_t` from a `sock`. Initializes all values in conn_tuple_t to `0`. Returns 1 success, 0 otherwise.
 */
static __always_inline int read_conn_tuple(conn_tuple_t* t, struct sock* skp, u64 pid_tgid, metadata_mask_t type) {
    __builtin_memset(t, 0, sizeof(conn_tuple_t));
    return read_conn_tuple_partial(t, skp, pid_tgid, type);
}

SEC("kretprobe/inet_csk_accept")
int kretprobe__inet_csk_accept(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_RC(ctx);
    if (sk == NULL) {
        return 0;
    }

    u64 pid_tgid = bpf_get_current_pid_tgid();
    log_debug("kretprobe/inet_csk_accept: tgid: %u, pid: %u\n", pid_tgid >> 32, pid_tgid & 0xFFFFFFFF);

    conn_tuple_t t = {};
    if (!read_conn_tuple(&t, sk, pid_tgid, CONN_TYPE_TCP)) {
        return 0;
    }
    handle_tcp_stats(&t, sk);
    handle_message(&t, 0, 0, CONN_DIRECTION_INCOMING, 0, 0, PACKET_COUNT_NONE);

    port_binding_t pb = {};
    pb.netns = t.netns;
    pb.port = t.sport;
    __u8 state = PORT_LISTENING;
    bpf_map_update_elem(&port_bindings, &pb, &state, BPF_NOEXIST);

    log_debug("kretprobe/inet_csk_accept: netns: %u, sport: %u, dport: %u\n", t.netns, t.sport, t.dport);
    return 0;
}

SEC("kprobe/inet_csk_listen_stop")
int kprobe__inet_csk_listen_stop(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    __u16 lport = read_sport(sk);
    if (lport == 0) {
        log_debug("ERR(inet_csk_listen_stop): lport is 0 \n");
        return 0;
    }

    port_binding_t t = {};
    t.netns = get_netns_from_sock(sk);
    t.port = lport;
    bpf_map_delete_elem(&port_bindings, &t);

    log_debug("kprobe/inet_csk_listen_stop: net ns: %u, lport: %u\n", t.netns, t.port);
    return 0;
}

#endif
