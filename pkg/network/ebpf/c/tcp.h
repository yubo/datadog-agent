#ifndef __TCP_H
#define __TCP_H

#include "tracer.h"
#include "tracer-stats.h"

#include "defs.h"
#include "sockfd.h"
#include "conn.h"

#include <linux/kconfig.h>
#include <net/inet_sock.h>
#include <uapi/linux/tcp.h>

static __always_inline __u64 offset_rtt() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_rtt", val);
    return val;
}

static __always_inline __u64 offset_rtt_var() {
    __u64 val = 0;
    LOAD_CONSTANT("offset_rtt_var", val);
    return val;
}

static __always_inline void handle_tcp_stats(conn_tuple_t* t, struct sock* sk) {
    u32 rtt = 0;
    u32 rtt_var = 0;
    bpf_probe_read(&rtt, sizeof(rtt), ((char*)sk) + offset_rtt());
    bpf_probe_read(&rtt_var, sizeof(rtt_var), ((char*)sk) + offset_rtt_var());

    tcp_stats_t stats = { .retransmits = 0, .rtt = rtt, .rtt_var = rtt_var };
    update_tcp_stats(t, stats);
}
static __always_inline void get_tcp_segment_counts(struct sock* skp, __u32* packets_in, __u32* packets_out) {
    // counting segments/packets not currently supported on prebuilt
    // to implement, would need to do the offset-guess on the following
    // fields in the tcp_sk: packets_in & packets_out (respectively)
    *packets_in = 0;
    *packets_out = 0;
}

SEC("kprobe/tcp_sendmsg")
int kprobe__tcp_sendmsg(struct pt_regs* ctx) {
    __u32 packets_in = 0;
    __u32 packets_out = 0;
    struct sock* skp = (struct sock*)PT_REGS_PARM1(ctx);
    size_t size = (size_t)PT_REGS_PARM3(ctx);
    u64 pid_tgid = bpf_get_current_pid_tgid();
    log_debug("kprobe/tcp_sendmsg: pid_tgid: %d, size: %d\n", pid_tgid, size);

    conn_tuple_t t = {};
    if (!read_conn_tuple(&t, skp, pid_tgid, CONN_TYPE_TCP)) {
        return 0;
    }

    handle_tcp_stats(&t, skp);
    get_tcp_segment_counts(skp, &packets_in, &packets_out);
    return handle_message(&t, size, 0, CONN_DIRECTION_UNKNOWN, packets_out, packets_in, PACKET_COUNT_ABSOLUTE);
}

SEC("kprobe/tcp_sendmsg/pre_4_1_0")
int kprobe__tcp_sendmsg__pre_4_1_0(struct pt_regs* ctx) {
    __u32 packets_in = 0;
    __u32 packets_out = 0;

    struct sock* sk = (struct sock*)PT_REGS_PARM2(ctx);
    size_t size = (size_t)PT_REGS_PARM4(ctx);
    u64 pid_tgid = bpf_get_current_pid_tgid();
    log_debug("kprobe/tcp_sendmsg/pre_4_1_0: pid_tgid: %d, size: %d\n", pid_tgid, size);

    conn_tuple_t t = {};
    if (!read_conn_tuple(&t, sk, pid_tgid, CONN_TYPE_TCP)) {
        return 0;
    }

    handle_tcp_stats(&t, sk);
    get_tcp_segment_counts(sk, &packets_in, &packets_out);
    return handle_message(&t, size, 0, CONN_DIRECTION_UNKNOWN, packets_out, packets_in, PACKET_COUNT_ABSOLUTE);
}

SEC("kprobe/tcp_cleanup_rbuf")
int kprobe__tcp_cleanup_rbuf(struct pt_regs* ctx) {

    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    int copied = (int)PT_REGS_PARM2(ctx);
    if (copied < 0) {
        return 0;
    }
    u64 pid_tgid = bpf_get_current_pid_tgid();
    log_debug("kprobe/tcp_cleanup_rbuf: pid_tgid: %d, copied: %d\n", pid_tgid, copied);

    conn_tuple_t t = {};
    if (!read_conn_tuple(&t, sk, pid_tgid, CONN_TYPE_TCP)) {
        return 0;
    }

    return handle_message(&t, 0, copied, CONN_DIRECTION_UNKNOWN, 0, 0, PACKET_COUNT_NONE);
}

SEC("kprobe/tcp_close")
int kprobe__tcp_close(struct pt_regs* ctx) {
    struct sock* sk;
    conn_tuple_t t = {};
    u64 pid_tgid = bpf_get_current_pid_tgid();
    sk = (struct sock*)PT_REGS_PARM1(ctx);

    clear_sockfd_maps(sk);

    // Get network namespace id
    log_debug("kprobe/tcp_close: tgid: %u, pid: %u\n", pid_tgid >> 32, pid_tgid & 0xFFFFFFFF);
    if (!read_conn_tuple(&t, sk, pid_tgid, CONN_TYPE_TCP)) {
        return 0;
    }
    log_debug("kprobe/tcp_close: netns: %u, sport: %u, dport: %u\n", t.netns, t.sport, t.dport);

    cleanup_conn(&t);
    return 0;
}

SEC("kretprobe/tcp_close")
int kretprobe__tcp_close(struct pt_regs* ctx) {
    flush_conn_close_if_full(ctx);
    return 0;
}


SEC("kprobe/tcp_retransmit_skb")
int kprobe__tcp_retransmit_skb(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    int segs = (int)PT_REGS_PARM3(ctx);
    log_debug("kprobe/tcp_retransmit\n");

    return handle_retransmit(sk, segs);
}

SEC("kprobe/tcp_retransmit_skb/pre_4_7_0")
int kprobe__tcp_retransmit_skb_pre_4_7_0(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    log_debug("kprobe/tcp_retransmit/pre_4_7_0\n");

    return handle_retransmit(sk, 1);
}

SEC("kprobe/tcp_set_state")
int kprobe__tcp_set_state(struct pt_regs* ctx) {
    u8 state = (u8)PT_REGS_PARM2(ctx);

    // For now we're tracking only TCP_ESTABLISHED
    if (state != TCP_ESTABLISHED) {
        return 0;
    }

    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    u64 pid_tgid = bpf_get_current_pid_tgid();
    conn_tuple_t t = {};
    if (!read_conn_tuple(&t, sk, pid_tgid, CONN_TYPE_TCP)) {
        return 0;
    }

    tcp_stats_t stats = { .state_transitions = (1 << state) };
    update_tcp_stats(&t, stats);

    return 0;
}

#endif
