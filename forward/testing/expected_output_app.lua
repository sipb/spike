local P = require("core.packet")
local ffi = require("ffi")
local L = require("core.link")
local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")

require("networking_magic_numbers")

local ExpectedOutputApp = {}

function extract_backend_addr(datagram)
   local eth_header = datagram:parse_match(Ethernet)
   if not eth_header then
      return nil, nil, "Bad Ethernet header."
   end
   local l3_type = eth_header:type()

   local backend_ip_len
   if l3_type == L3_IPV4 then
      backend_ip_len = 4
   elseif l3_type == L3_IPV6 then
      backend_ip_len = 16
   else
      return nil, nil,
         "Unknown EtherType when extracting backend IP address."
   end

   local ip_class = eth_header:upper_layer()
   local ip_header = datagram:parse_match(ip_class)
   if not ip_header then
      return nil, nil, "Bad IP header."
   end
   local backend_ip = ip_header:dst()

   -- Unparse IP and Ethernet
   datagram:unparse(2)
   return backend_ip, backend_ip_len
end

-- Arguments:
-- synthesis (PacketSynthesisContext) -- The packet synthesis context for
--    the current session.
function ExpectedOutputApp:new(opts)
   return setmetatable({
      synthesis = opts.synthesis,
      backends = nil,
      curr_packet_index = 1,
      expected_output_generators = nil,
      err = nil
   }, {
      __index = ExpectedOutputApp
   })
end

function ExpectedOutputApp:push()
   local i = assert(self.input.input, "input port not found")
   local o = assert(self.output.output, "output port not found")
   while not L.empty(i) do
      self:process_packet(i, o)
   end
end

function ExpectedOutputApp:init(generators, backends)
    self.curr_packet_index = 1
    self.expected_output_generators = generators
    self.backends = backends
end

function ExpectedOutputApp:register_error(error_message)
   if self.err then return end
   self.err = error_message
end

function ExpectedOutputApp:process_packet(i, o)
   local p = L.receive(i)

   -- Note: This would be extended in the future to allow other types of
   -- expected output.
   local datagram = Datagram:new(p)
   local backend_addr, backend_addr_len, err =
      extract_backend_addr(datagram)
   if err then
      self:register_error(err)
      P.free(p)
      return
   end

   local match_found = false
   for _, b in ipairs(self.backends) do
      -- TODO: Extend to support IPv6 backends
      if backend_addr_len == b.addr_len and
         ffi.string(backend_addr, backend_addr_len) ==
         ffi.string(b.addr, b.addr_len) then
         match_found = true
         break
      end
   end
   if not match_found then
      local backend_addr_str
      if backend_addr_len == 4 then
         backend_addr_str = IPV4:ntop(backend_addr)
      elseif backend_addr_len == 16 then
         backend_addr_str = IPV6:ntop(backend_addr)
      else
         self:register_error("Unknown backend address length.")
         return
      end
      self:register_error(
         "Packet forwarded to bad backend address: "..backend_addr_str)
      return
   end

   local expected_packet =
      self.expected_output_generators[self.curr_packet_index](backend_addr)
   self.curr_packet_index = self.curr_packet_index + 1
   P.free(p)

   link.transmit(o, expected_packet)
end

return ExpectedOutputApp
