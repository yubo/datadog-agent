#ifndef __IPV6_H
#define __IPV6_H

#include "defs.h"

static __always_inline bool are_fl6_offsets_known() {
    __u64 val = 0;
    LOAD_CONSTANT("fl6_offsets", val);
    return val == ENABLED;
}

static __always_inline __u64 offset_saddr_fl6() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_saddr_fl6", val);
    return val;
}

static __always_inline __u64 offset_daddr_fl6() {
     __u64 val = 0;
     LOAD_CONSTANT("offset_daddr_fl6", val);
     return val;
}

static __always_inline __u64 offset_sport_fl6() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_sport_fl6", val);
    return val;
}

static __always_inline __u64 offset_dport_fl6() {
     __u64 val = 0;
     LOAD_CONSTANT("offset_dport_fl6", val);
     return val;
}

/* check if IPs are IPv4 mapped to IPv6 ::ffff:xxxx:xxxx
 * https://tools.ietf.org/html/rfc4291#section-2.5.5
 * the addresses are stored in network byte order so IPv4 adddress is stored
 * in the most significant 32 bits of part saddr_l and daddr_l.
 * Meanwhile the end of the mask is stored in the least significant 32 bits.
 */
static __always_inline bool is_ipv4_mapped_ipv6(__u64 saddr_h, __u64 saddr_l, __u64 daddr_h, __u64 daddr_l) {
#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__
    return ((saddr_h == 0 && ((__u32)saddr_l == 0xFFFF0000)) || (daddr_h == 0 && ((__u32)daddr_l == 0xFFFF0000)));
#elif __BYTE_ORDER__ == __ORDER_BIG_ENDIAN__
    return ((saddr_h == 0 && ((__u32)(saddr_l >> 32) == 0x0000FFFF)) || (daddr_h == 0 && ((__u32)(daddr_l >> 32) == 0x0000FFFF)));
#else
#error "Fix your compiler's __BYTE_ORDER__?!"
#endif
}

static __always_inline void read_in6_addr(u64 *addr_h, u64 *addr_l, const struct in6_addr *in6) {
    bpf_probe_read(addr_h, sizeof(u64), (void *)&(in6->in6_u.u6_addr32[0]));
    bpf_probe_read(addr_l, sizeof(u64), (void *)&(in6->in6_u.u6_addr32[2]));
}

static __always_inline int handle_ip6_skb(struct sock* sk, size_t size, struct flowi6* fl6) {
    u64 pid_tgid = bpf_get_current_pid_tgid();
    size = size - sizeof(struct udphdr);

    conn_tuple_t t = {};
    if (!read_conn_tuple(&t, sk, pid_tgid, CONN_TYPE_UDP)) {
        if (!are_fl6_offsets_known()) {
            log_debug("ERR: src/dst addr not set, fl6 offsets are not known\n");
            increment_telemetry_count(udp_send_missed);
            return 0;
        }
        read_in6_addr(&t.saddr_h, &t.saddr_l, (struct in6_addr*)(((char*)fl6) + offset_saddr_fl6()));
        read_in6_addr(&t.daddr_h, &t.daddr_l, (struct in6_addr*)(((char*)fl6) + offset_daddr_fl6()));

        if (!(t.saddr_h || t.saddr_l)) {
            log_debug("ERR(fl6): src addr not set src_l:%d,src_h:%d\n", t.saddr_l, t.saddr_h);
            increment_telemetry_count(udp_send_missed);
            return 0;
        }
        if (!(t.daddr_h || t.daddr_l)) {
            log_debug("ERR(fl6): dst addr not set dst_l:%d,dst_h:%d\n", t.daddr_l, t.daddr_h);
            increment_telemetry_count(udp_send_missed);
            return 0;
        }

        // Check if we can map IPv6 to IPv4
        if (is_ipv4_mapped_ipv6(t.saddr_h, t.saddr_l, t.daddr_h, t.daddr_l)) {
            t.metadata |= CONN_V4;
            t.saddr_h = 0;
            t.daddr_h = 0;
            t.saddr_l = (u32)(t.saddr_l >> 32);
            t.daddr_l = (u32)(t.daddr_l >> 32);
        } else {
            t.metadata |= CONN_V6;
        }

        bpf_probe_read(&t.sport, sizeof(t.sport), ((char*)fl6) + offset_sport_fl6());
        bpf_probe_read(&t.dport, sizeof(t.dport), ((char*)fl6) + offset_dport_fl6());

        if (t.sport == 0 || t.dport == 0) {
            log_debug("ERR(fl6): src/dst port not set: src:%d, dst:%d\n", t.sport, t.dport);
            increment_telemetry_count(udp_send_missed);
            return 0;
        }

        t.sport = ntohs(t.sport);
        t.dport = ntohs(t.dport);
    }

    log_debug("kprobe/ip6_make_skb: pid_tgid: %d, size: %d\n", pid_tgid, size);
    handle_message(&t, size, 0, CONN_DIRECTION_UNKNOWN, 0, 0, PACKET_COUNT_NONE);
    increment_telemetry_count(udp_send_processed);

    return 0;
}

// commit: https://github.com/torvalds/linux/commit/26879da58711aa604a1b866cbeedd7e0f78f90ad
// changed the arguments to ip6_make_skb and introduced the struct ipcm6_cookie
SEC("kprobe/ip6_make_skb/pre_4_7_0")
int kprobe__ip6_make_skb__pre_4_7_0(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    size_t size = (size_t)PT_REGS_PARM4(ctx);
    struct flowi6* fl6 = (struct flowi6*)PT_REGS_PARM9(ctx);

    return handle_ip6_skb(sk, size, fl6);
}

SEC("kprobe/ip6_make_skb")
int kprobe__ip6_make_skb(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    size_t size = (size_t)PT_REGS_PARM4(ctx);
    struct flowi6* fl6 = (struct flowi6*)PT_REGS_PARM7(ctx);

    return handle_ip6_skb(sk, size, fl6);
}

#endif
