const std = @import("std");

pub fn main() void{
    std.debug.print("Hello, {s}\n",.{"World"});
    
    const a: i32 = 5;
    const b:i32 = 5000;

    std.debug.print("VAL: {d}\n",.{a+b});

    const c = [5]i32{1,2,3,4,5};
    const d = [_]i32{};
    
    std.debug.print("LEN c: {d}\n",.{c.len});
    std.debug.print("LEN d: {d}\n",.{d.len});
}
