local ffi = require("ffi")
local L = require("core.link")
local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")

local ExpectedOutputApp = {}

local function extract_backend_addr(datagram)
   local eth_header = datagram:parse_match(Ethernet)
   assert(eth_header, "Bad Ethernet header.")
   local ip_class = eth_header:upper_layer()
   local ip_header = datagram:parse_match(ip_class)
   assert(ip_header, "Bad IP header.")
   local backend_ip = ip_header:dst()
   -- Unparse IP and Ethernet
   datagram:unparse(2)
   return backend_ip
end

-- Arguments:
-- synthesis (PacketSynthesisContext) -- The packet synthesis context for
--    the current session.
function ExpectedOutputApp:new(opts)
   return setmetatable({
      synthesis = opts.synthesis,
      valid_backend_addrs = nil,
      curr_packet_index = 1,
      expected_output_generators = nil
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

function ExpectedOutputApp:init(generators, valid_backend_addrs)
    self.curr_packet_index = 1
    self.expected_output_generators = generators
    self.valid_backend_addrs = valid_backend_addrs
end

function ExpectedOutputApp:process_packet(i, o)
   local p = L.receive(i)

   -- Note: This would be extended in the future to allow other types of
   -- expected output.
   local datagram = Datagram:new(p)
   local backend_addr = extract_backend_addr(datagram)

   local match_found = false
   for _, addr in ipairs(self.valid_backend_addrs) do
      if ffi.string(backend_addr) == ffi.string(addr) then
         match_found = true
         break
      end
   end
   if not match_found then
      error("Packet forwarded to bad backend address: "..
         IPV4:ntop(backend_addr))
   end

   local expected_packet =
      self.expected_output_generators[self.curr_packet_index](backend_addr)
   self.curr_packet_index = self.curr_packet_index + 1

   link.transmit(o, expected_packet)
end

return ExpectedOutputApp
