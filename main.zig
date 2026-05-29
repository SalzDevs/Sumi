const std = @import("std");

pub fn main() void{
    std.debug.print("Hello, {s}\n",.{"World"});

    const a: i32 = 5;
    const b:i32 = 5000;

    std.debug.print("VAL: {d}\n",.{a+b});
}
