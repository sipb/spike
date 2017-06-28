local P = require("core.packet")
local L = require("core.link")
local G = require("lib.protocol.gre")
local D = require("lib.protocol.datagram")

local M = {}

_G._NAME = "gre_wrap"  -- Snabb requires this for some reason


GRE_wrap = {}

function GRE_wrap:new(protocol)
   return setmetatable({protocol = protocol}, {__index = GRE_wrap})
end

function GRE_wrap:push()
   local i = assert(self.input.input, "input port not found")
   local o = assert(self.output.output, "output port not found")
   while not L.empty(i) do
      self:process_packet(i, o)
   end
end

function GRE_wrap:process_packet(i, o)
   local p = L.receive(i)
   local gre_header = G:new({protocol = self.protocol})
   -- TODO figure out what the encapsulated protocol should actually be
   local datagram = D:new(p, self.protocol, {delayed_commit = false})
   datagram:push(gre_header)

   link.transmit(o, datagram:packet())
end

M.GRE_wrap = GRE_wrap


return M
