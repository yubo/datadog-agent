#ifndef _GOROUTINE_TRACKER_H
#define _GOROUTINE_TRACKER_H

#include "span.h"

struct goroutine_tracker_event_t {
    struct kevent_t event;
    struct process_context_t process;
    struct container_context_t container;
};

int __attribute__((always_inline)) handle_goroutine_tracker(struct pt_regs *ctx) {
    // send an event to user space to request the activation of the goroutine tracker on the current process
    struct goroutine_tracker_event_t event = {};
    struct proc_cache_t *entry = fill_process_context(&event.process);
    fill_container_context(entry, &event.container);

    send_event(ctx, EVENT_GOROUTINE_TRACKER, event);
    return 0;
}

SEC("uprobe/runtime.execute")
int uprobe_readline(struct pt_regs *ctx)
{
    void *goroutine;
    u64 goroutine_id, offset;
    u64 id = bpf_get_current_pid_tgid();
    u32 co_ctx_key = id >> 32;

    // fetch the pointer to the goroutine about to be scheduled on the current thread
    bpf_probe_read(&goroutine, sizeof(goroutine), (void *) PT_REGS_SP(ctx) + 8);
    if (goroutine == NULL) {
        return 0;
    }

    // fetch the coroutine data for the current thread, it contains the offset used to dereference the goroutine id
    struct coroutine_ctx_t *co_ctx = bpf_map_lookup_elem(&coroutine_ctx, &co_ctx_key);
    if (co_ctx == NULL) {
        return 0;
    }

    // parse offset
    bpf_probe_read(&offset, sizeof(offset), co_ctx->data);

    // fetch goroutine id
    bpf_probe_read(&goroutine_id, sizeof(goroutine_id), (void *) goroutine + offset);

    // update the thread id <-> coroutine id mapping
    bpf_map_update_elem(&coroutine_ids, &id, &goroutine_id, BPF_ANY);
    return 0;
}

#endif
