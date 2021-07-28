#include "tracer.h"

#include "tracer-events.h"
#include "tracer-maps.h"
#include "tracer-stats.h"
#include "tracer-telemetry.h"
#include "sockfd.h"

#include "bpf_helpers.h"
#include "bpf_endian.h"
#include "ip.h"
#include "ipv6.h"
#include "defs.h"
#include "netns.h"
#include "conn.h"
#include "tcp.h"
#include "ipv6.h"
#include "ip.h"
#include "udp.h"
#include "tcp.h"
#include "inet.h"
#include "sendfile.h"
#include "dns.h"

#include <linux/kconfig.h>
#include <net/inet_sock.h>
#include <net/net_namespace.h>
#include <net/tcp_states.h>
#include <uapi/linux/ip.h>
#include <uapi/linux/ipv6.h>
#include <uapi/linux/ptrace.h>
#include <uapi/linux/tcp.h>
#include <uapi/linux/udp.h>

// This number will be interpreted by elf-loader to set the current running kernel version
__u32 _version SEC("version") = 0xFFFFFFFE; // NOLINT(bugprone-reserved-identifier)

char _license[] SEC("license") = "GPL"; // NOLINT(bugprone-reserved-identifier)
