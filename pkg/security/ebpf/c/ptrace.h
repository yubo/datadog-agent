#ifndef _PTRACE_H_
#define _PTRACE_H_

struct ptrace_event_t {
    struct kevent_t event;
    struct process_context_t process;
    struct container_context_t container;
    struct syscall_t syscall;
    u32 request;
    u32 pid;
};

int __attribute__((always_inline)) trace__ptrace(u32 request, u32 pid) {
    struct syscall_cache_t syscall = {
        .type = EVENT_PTRACE,
        .ptrace = {
            .request = request,
            .pid = pid,
        }
    };

    cache_syscall(&syscall);

    return 0;
}

SYSCALL_KPROBE2(ptrace, u32, request, pid_t, pid) {
    return trace__ptrace(request, (u32)pid);
}

int __attribute__((always_inline)) trace__ptrace_ret(struct pt_regs *ctx) {
    struct syscall_cache_t *syscall = pop_syscall();
    if (!syscall)
        return 0;

    struct ptrace_event_t event = {
        .event.type = EVENT_PTRACE,
        .syscall = {
            .retval = (int)PT_REGS_RC(ctx),
            .timestamp = bpf_ktime_get_ns(),
        },
        .request = syscall->ptrace.request,
        .pid = syscall->ptrace.pid,
    };

    struct proc_cache_t *entry = fill_process_data(&event.process);
    fill_container_data(entry, &event.container);

    send_event(ctx, event);

    return 0;
}

SYSCALL_KRETPROBE(ptrace)
{
    return trace__ptrace_ret(ctx);
}

#endif
