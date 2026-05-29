const std = @import("std");

fn addFive(x: u32) u32{
    return x+5;
}

fn fibonacci(n: u16) u16{
    if (n==0 or n==1) return n;
    return fibonacci(n-1) + fibonacci(n-2);
}

fn addXtoY(x:i16,y:i16) i16{
    return x+y;
}

pub fn main() void{
    var x:i16 = 16;
    const y:i16 = 4;
    if  (true){
        defer x = addXtoY(x,y);
        std.debug.print("x before defer:{d}\n",.{x});
    }
    std.debug.print("x after defer:{d}\n",.{x});
}
