const std = @import("std");
const expect = std.testing.expect;

const FileOpenError = error{
    AccessDenied,
    OutOfMemory,
    FileNotFound,
};

test "coerce error from a subset to a superset"{
    const AllocationError = error{OutOfMemory};
    const err: FileOpenError = AllocationError.OutOfMemory;
    try expect(err==FileOpenError.OutOfMemory);
}

pub fn main() void{
 std.print.debug("Hello {s}\n",.{"Seamen"});
}
