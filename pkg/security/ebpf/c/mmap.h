#ifndef _MMAP_H_
#define _MMAP_H_

struct mmap_event_t {
    struct kevent_t event;
    struct process_context_t process;
    struct container_context_t container;
    struct syscall_t syscall;
    u64 addr;
    u32 len;
    int protection;
};

int __attribute__((always_inline)) trace__mmap(void * addr, size_t len, int protection) {
    // We only care about memory regions with both VM_WRITE and VM_EXEC activated
    if ((protection & (VM_WRITE|VM_EXEC)) != (VM_WRITE|VM_EXEC)) {
        return 0;
    }

    struct syscall_cache_t syscall = {
        .type = EVENT_MMAP,
        .mmap = {
            .addr = (u64)addr,
            .len = (u32)len,
            .protection = protection,
        }
    };

    cache_syscall(&syscall);
    return 0;
}

SYSCALL_KPROBE3(mmap, void *, addr, size_t, len, int, protection) {
    return trace__mmap(addr, len, protection);
}

int __attribute__((always_inline)) trace__mmap_ret(struct pt_regs *ctx) {
    struct syscall_cache_t *syscall = pop_syscall();
    if (!syscall)
        return 0;

    struct mmap_event_t event = {
        .event.type = EVENT_MMAP,
        .syscall = {
            .timestamp = bpf_ktime_get_ns(),
        },
        .addr = (u64)PT_REGS_RC(ctx),
        .len = syscall->mmap.len,
        .protection = syscall->mmap.protection,
    };

    struct proc_cache_t *entry = fill_process_data(&event.process);
    fill_container_data(entry, &event.container);
    send_event(ctx, event);
    return 0;
}

SYSCALL_KRETPROBE(mmap)
{
    return trace__mmap_ret(ctx);
}

#endif
