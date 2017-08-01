-- LuaJIT's FFI doesn't have a preprocessor, so we have to include the
-- contents of the cgo header ourselves.  Make sure to keep this up to
-- date with the header file from cgo.

-- TODO come up with a better solution than manual preprocessing
local ffi = require("ffi")

ffi.cdef[[
/* Created by "go tool cgo" - DO NOT EDIT. */

/* package github.com/sipb/spike */

/* Start of preamble from import "C" comments.  */




/* End of preamble from import "C" comments.  */


/* Start of boilerplate cgo prologue.  */


typedef signed char GoInt8;
typedef unsigned char GoUint8;
typedef short GoInt16;
typedef unsigned short GoUint16;
typedef int GoInt32;
typedef unsigned int GoUint32;
typedef long long GoInt64;
typedef unsigned long long GoUint64;
typedef GoInt64 GoInt;
typedef GoUint64 GoUint;
//typedef __SIZE_TYPE__ GoUintptr;
typedef float GoFloat32;
typedef double GoFloat64;
typedef float _Complex GoComplex64;
typedef double _Complex GoComplex128;

/*
  static assertion to make sure the file is being used on architecture
  at least with matching size of GoInt.
*/
typedef char _check_for_64_bit_pointer_matching_GoInt[sizeof(void*)==64/8 ? 1:-1];

typedef struct { const char *p; GoInt n; } GoString;
typedef void *GoMap;
typedef void *GoChan;
typedef struct { void *t; void *v; } GoInterface;
typedef struct { void *data; GoInt len; GoInt cap; } GoSlice;


/* End of boilerplate cgo prologue.  */



extern void Init();

extern void AddBackend(GoString p0, GoSlice p1);

extern void RemoveBackend(GoString p0);

/* Return type for Lookup */
struct Lookup_return {
	GoSlice r0;
	GoUint8 r1;
};

extern struct Lookup_return Lookup(GoSlice p0);
]]

local golib = ffi.load(os.getenv("LIBSPIKE"))
local GoString = ffi.typeof("GoString")
local GoSlice = ffi.typeof("GoSlice")
local GoInt = ffi.typeof("GoInt")

local M = {}

function M.Init()
   return golib.Init()
end

function M.AddBackend(service, ip, ip_len)
   return golib.AddBackend(GoString(service, #service),
                           GoSlice(ip, ip_len, ip_len))
end

function M.RemoveBackend(service)
   return golib.RemoveBackend(GoString(service, #service))
end

function M.Lookup(x, x_len)
   local ret = golib.Lookup(GoSlice(x, x_len, x_len))
   return ret.r0.data, ret.r0.len, ret.r1
end

return M
