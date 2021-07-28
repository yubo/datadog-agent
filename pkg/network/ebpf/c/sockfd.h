#ifndef __SOCKFD_H
#define __SOCKFD_H

#include "defs.h"
#include "tracer.h"
#include <linux/types.h>

static __always_inline __u64 offset_socket_sk() {
     __u64 val = 0;
     LOAD_CONSTANT("offset_socket_sk", val);
     return val;
}

typedef struct {
    __u32 pid;
    __u32 fd;
} pid_fd_t;

// This map is used to to temporarily store function arguments (sockfd) for
// sockfd_lookup_light function calls, so they can be acessed by the corresponding kretprobe.
// * Key is the pid_tgid;
// * Value the socket FD;
struct bpf_map_def SEC("maps/sockfd_lookup_args") sockfd_lookup_args = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(__u64),
    .value_size = sizeof(__u32),
    .max_entries = 1024,
    .pinning = 0,
    .namespace = "",
};

struct bpf_map_def SEC("maps/sock_by_pid_fd") sock_by_pid_fd = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(pid_fd_t),
    .value_size = sizeof(struct sock*),
    .max_entries = 1024,
    .pinning = 0,
    .namespace = "",
};

struct bpf_map_def SEC("maps/pid_fd_by_sock") pid_fd_by_sock = {
    .type = BPF_MAP_TYPE_HASH,
    .key_size = sizeof(struct sock*),
    .value_size = sizeof(pid_fd_t),
    .max_entries = 1024,
    .pinning = 0,
    .namespace = "",
};

static __always_inline void clear_sockfd_maps(struct sock* sock) {
    if (sock == NULL) {
        return;
    }

    pid_fd_t* pid_fd = bpf_map_lookup_elem(&pid_fd_by_sock, &sock);
    if (pid_fd == NULL) {
        return;
    }

    // Copy map value to stack before re-using it (needed for Kernel 4.4)
    pid_fd_t pid_fd_copy = {};
    __builtin_memcpy(&pid_fd_copy, pid_fd, sizeof(pid_fd_t));
    pid_fd = &pid_fd_copy;

    bpf_map_delete_elem(&sock_by_pid_fd, pid_fd);
    bpf_map_delete_elem(&pid_fd_by_sock, &sock);
}

SEC("kprobe/sockfd_lookup_light")
int kprobe__sockfd_lookup_light(struct pt_regs* ctx) {
    int sockfd = (int)PT_REGS_PARM1(ctx);
    u64 pid_tgid = bpf_get_current_pid_tgid();

    // Check if have already a map entry for this pid_fd_t
    pid_fd_t key = {
        .pid = pid_tgid >> 32,
        .fd = sockfd,
    };
    struct sock** sock = bpf_map_lookup_elem(&sock_by_pid_fd, &key);
    if (sock != NULL) {
        return 0;
    }

    bpf_map_update_elem(&sockfd_lookup_args, &pid_tgid, &sockfd, BPF_ANY);
    return 0;
}

// this kretprobe is essentially creating:
// * an index of pid_fd_t to a struct sock*;
// * an index of struct sock* to pid_fd_t;
SEC("kretprobe/sockfd_lookup_light")
int kretprobe__sockfd_lookup_light(struct pt_regs* ctx) {
    u64 pid_tgid = bpf_get_current_pid_tgid();
    int *sockfd = bpf_map_lookup_elem(&sockfd_lookup_args, &pid_tgid);
    if (sockfd == NULL) {
        return 0;
    }

    // For now let's only store information for TCP sockets
    struct socket* socket = (struct socket*)PT_REGS_RC(ctx);
    enum sock_type sock_type = 0;
    bpf_probe_read(&sock_type, sizeof(short), &socket->type);
    if (sock_type != SOCK_STREAM) {
        goto cleanup;
    }

    // Retrieve struct sock* pointer from struct socket*
    struct sock *sock = NULL;
    bpf_probe_read(&sock, sizeof(sock), (char*)socket + offset_socket_sk());

    pid_fd_t pid_fd = {
        .pid = pid_tgid >> 32,
        .fd = (*sockfd),
    };

    // These entries are cleaned up by tcp_close
    bpf_map_update_elem(&pid_fd_by_sock, &sock, &pid_fd, BPF_ANY);
    bpf_map_update_elem(&sock_by_pid_fd, &pid_fd, &sock, BPF_ANY);
cleanup:
    bpf_map_delete_elem(&sockfd_lookup_args, &pid_tgid);
    return 0;
}

#endif
