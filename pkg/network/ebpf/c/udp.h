#ifndef __UDP_H
#define __UDP_H

// We can only get the accurate number of copied bytes from the return value, so we pass our
// sock* pointer from the kprobe to the kretprobe via a map (udp_recv_sock) to get all required info
//
// The same issue exists for TCP, but we can conveniently use the downstream function tcp_cleanup_rbuf
//
// On UDP side, no similar function exists in all kernel versions, though we may be able to use something like
// skb_consume_udp (v4.10+, https://elixir.bootlin.com/linux/v4.10/source/net/ipv4/udp.c#L1500)
SEC("kprobe/udp_recvmsg")
int kprobe__udp_recvmsg(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    struct msghdr* msg = (struct msghdr*)PT_REGS_PARM2(ctx);
    int flags = (int)PT_REGS_PARM5(ctx);
    log_debug("kprobe/udp_recvmsg: flags: %x\n", flags);
    if (flags & MSG_PEEK) {
        return 0;
    }

    u64 pid_tgid = bpf_get_current_pid_tgid();
    udp_recv_sock_t t = { .sk = NULL, .msg = NULL };
    if (sk) {
        bpf_probe_read(&t.sk, sizeof(t.sk), &sk);
    }
    if (msg) {
        bpf_probe_read(&t.msg, sizeof(t.msg), &msg);
    }

    bpf_map_update_elem(&udp_recv_sock, &pid_tgid, &t, BPF_ANY);
    return 0;
}

SEC("kprobe/udp_recvmsg/pre_4_1_0")
int kprobe__udp_recvmsg_pre_4_1_0(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_PARM2(ctx);
    struct msghdr* msg = (struct msghdr*)PT_REGS_PARM3(ctx);
    int flags = (int)PT_REGS_PARM6(ctx);
    log_debug("kprobe/udp_recvmsg: flags: %x\n", flags);
    if (flags & MSG_PEEK) {
        return 0;
    }

    u64 pid_tgid = bpf_get_current_pid_tgid();
    udp_recv_sock_t t = { .sk = NULL, .msg = NULL };
    if (sk) {
        bpf_probe_read(&t.sk, sizeof(t.sk), &sk);
    }
    if (msg) {
        bpf_probe_read(&t.msg, sizeof(t.msg), &msg);
    }

    bpf_map_update_elem(&udp_recv_sock, &pid_tgid, &t, BPF_ANY);
    return 0;
}

SEC("kretprobe/udp_recvmsg")
int kretprobe__udp_recvmsg(struct pt_regs* ctx) {
    u64 pid_tgid = bpf_get_current_pid_tgid();

    // Retrieve socket pointer from kprobe via pid/tgid
    udp_recv_sock_t* st = bpf_map_lookup_elem(&udp_recv_sock, &pid_tgid);
    if (!st) { // Missed entry
        return 0;
    }

    // Make sure we clean up the key
    bpf_map_delete_elem(&udp_recv_sock, &pid_tgid);

    int copied = (int)PT_REGS_RC(ctx);
    if (copied < 0) { // Non-zero values are errors (or a peek) (e.g -EINVAL)
        log_debug("kretprobe/udp_recvmsg: ret=%d < 0, pid_tgid=%d\n", copied, pid_tgid);
        return 0;
    }

    log_debug("kretprobe/udp_recvmsg: ret=%d\n", copied);

    struct sockaddr * sa = NULL;
    if (st->msg) {
        bpf_probe_read(&sa, sizeof(sa), &(st->msg->msg_name));
    }

    conn_tuple_t t = {};
    __builtin_memset(&t, 0, sizeof(conn_tuple_t));
    sockaddr_to_addr(sa, &t.daddr_h, &t.daddr_l, &t.dport);

    if (!read_conn_tuple_partial(&t, st->sk, pid_tgid, CONN_TYPE_UDP)) {
        log_debug("ERR(kretprobe/udp_recvmsg): error reading conn tuple, pid_tgid=%d\n", pid_tgid);
        return 0;
    }

    log_debug("kretprobe/udp_recvmsg: pid_tgid: %d, return: %d\n", pid_tgid, copied);
    // segment count is not currently enabled on prebuilt.
    // to enable, change PACKET_COUNT_NONE => PACKET_COUNT_INCREMENT
    handle_message(&t, 0, copied, CONN_DIRECTION_UNKNOWN, 0, 1, PACKET_COUNT_NONE);

    return 0;
}

SEC("kprobe/udp_destroy_sock")
int kprobe__udp_destroy_sock(struct pt_regs* ctx) {
    struct sock* sk = (struct sock*)PT_REGS_PARM1(ctx);
    conn_tuple_t tup = {};
     u64 pid_tgid = bpf_get_current_pid_tgid();
    int valid_tuple = read_conn_tuple(&tup, sk, pid_tgid, CONN_TYPE_UDP);

    __u16 lport = 0;
    if (valid_tuple) {
        cleanup_conn(&tup);
        lport = tup.sport;
    } else {
        // get the port for the current sock
        lport = read_sport(sk);
    }

    if (lport == 0) {
        log_debug("ERR(udp_destroy_sock): lport is 0\n");
        return 0;
    }

    // although we have net ns info, we don't use it in the key
    // since we don't have it everywhere for udp port bindings
    // (see sys_enter_bind/sys_exit_bind below)
    port_binding_t t = {};
    t.netns = 0;
    t.port = lport;
    bpf_map_delete_elem(&udp_port_bindings, &t);

    log_debug("kprobe/udp_destroy_sock: port %d marked as closed\n", lport);

    return 0;
}

SEC("kretprobe/udp_destroy_sock")
int kretprobe__udp_destroy_sock(struct pt_regs * ctx) {
    flush_conn_close_if_full(ctx);
    return 0;
}

#endif
