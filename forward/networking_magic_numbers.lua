local ffi = require("ffi")

L3_IPV4 = 0x0800
L3_IPV6 = 0x86DD
L4_TCP = 0x06
L4_UDP = 0x11
L4_GRE = 0x2f

IP_HEADER_LENGTH = 20

IP_MF_FLAG = 0x1
IP_DF_FLAG = 0x2

ipv4_addr = ffi.typeof("unsigned char[4]")
ipv6_addr = ffi.typeof("unsigned char[16]")
ip_addr = {[L3_IPV4] = ipv4_addr,
           [L3_IPV6] = ipv6_addr}
ip_addr_len = {[L3_IPV4] = 4,
               [L3_IPV6] = 16}
