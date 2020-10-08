#ifndef _MPROTECT_H_
#define _MPROTECT_H_

struct mprotect_event_t {
    struct kevent_t event;
    struct process_context_t process;
    struct container_context_t container;
    struct syscall_t syscall;
    u64 vm_start;
    u64 vm_end;
    u64 vm_protection;
    u64 req_protection;
};

static __attribute__((always_inline)) int is_sensitive_change(u64 vm_protection, u64 req_protection) {
    if ((!(vm_protection & VM_EXEC)) && (req_protection & VM_EXEC)) {
        return 1;
    }
    if ((vm_protection & VM_EXEC) && !(vm_protection & VM_WRITE)
        && ((req_protection & (VM_WRITE|VM_EXEC)) == (VM_WRITE|VM_EXEC))) {
        return 1;
    }
    if (((vm_protection & (VM_WRITE|VM_EXEC)) == (VM_WRITE|VM_EXEC))
        && (req_protection & VM_EXEC) && !(req_protection & VM_WRITE)) {
        return 1;
    }
    return 0;
}

SEC("kprobe/security_file_mprotect")
int kprobe__security_file_mprotect(struct pt_regs *ctx) {
    // Retrieve vma information
    struct vm_area_struct *vma = (struct vm_area_struct *)PT_REGS_PARM1(ctx);
    u64 vm_protection;
    bpf_probe_read(&vm_protection, sizeof(vm_protection), &vma->vm_flags);
    u64 req_protection = (u64)PT_REGS_PARM2(ctx);

    if (!is_sensitive_change(vm_protection, req_protection)) {
        return 0;
    }

    struct mprotect_event_t event = {
        .event.type = EVENT_MPROTECT,
        .syscall = {
            .timestamp = bpf_ktime_get_ns(),
        },
        .vm_protection = vm_protection,
        .req_protection = req_protection,
    };
    bpf_probe_read(&event.vm_start, sizeof(event.vm_start), &vma->vm_start);
    bpf_probe_read(&event.vm_end, sizeof(event.vm_end), &vma->vm_end);

    struct proc_cache_t *entry = fill_process_data(&event.process);
    fill_container_data(entry, &event.container);
    send_event(ctx, event);
    return 0;
}

#endif
