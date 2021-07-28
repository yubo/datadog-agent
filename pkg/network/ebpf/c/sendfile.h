#ifndef __SENDFILE_H
#define __SENDFILE_H

SEC("kprobe/do_sendfile")
int kprobe__do_sendfile(struct pt_regs* ctx) {
    u32 fd_out = (int)PT_REGS_PARM1(ctx);
    u64 pid_tgid = bpf_get_current_pid_tgid();
    pid_fd_t key = {
        .pid = pid_tgid >> 32,
        .fd = fd_out,
    };
    struct sock** sock = bpf_map_lookup_elem(&sock_by_pid_fd, &key);
    if (sock == NULL) {
        return 0;
    }

    // bring map value to eBPF stack to satisfy Kernel 4.4 verifier
    struct sock* skp = *sock;
    bpf_map_update_elem(&do_sendfile_args, &pid_tgid, &skp, BPF_ANY);
    return 0;
}

SEC("kretprobe/do_sendfile")
int kretprobe__do_sendfile(struct pt_regs* ctx) {
    u64 pid_tgid = bpf_get_current_pid_tgid();
    struct sock** sock = bpf_map_lookup_elem(&do_sendfile_args, &pid_tgid);
    if (sock == NULL) {
        return 0;
    }

    conn_tuple_t t = {};
    if (!read_conn_tuple(&t, *sock, pid_tgid, CONN_TYPE_TCP)) {
        goto cleanup;
    }

    size_t sent = (size_t)PT_REGS_RC(ctx);
    handle_message(&t, sent, 0, CONN_DIRECTION_UNKNOWN, 0, 0, PACKET_COUNT_NONE);
cleanup:
    bpf_map_delete_elem(&do_sendfile_args, &pid_tgid);
    return 0;
}

#endif
