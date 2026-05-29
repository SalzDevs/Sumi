const std = @import("std");

fn addFive(x: u32) u32{
    return x+5;
}

fn fibonacci(n: u16) u16{
    if (n==0 or n==1) return n;
    return fibonacci(n-1) + fibonacci(n-2);
}

pub fn main() void{
    std.debug.print("Hello, {s}\n",.{"World"});
    
    const a: i32 = 5;
    const b:i32 = 5000;

    std.debug.print("VAL: {d}\n",.{a+b});

    const c = [5]i32{1,2,3,4,5};
    const d = [_]i32{};
    
    std.debug.print("LEN c: {d}\n",.{c.len});
    std.debug.print("LEN d: {d}\n",.{d.len});
    
    var  eval_flag = true;
    var if_true_inc:i32 = 0; 
    
    if_true_inc += if (eval_flag) 1 else 0;
    if_true_inc += if (eval_flag) 1 else 0;
    if_true_inc += if (eval_flag) 1 else 0;
    eval_flag = false;
    if_true_inc += if (eval_flag) 1 else 0;
    if_true_inc += if (eval_flag) 1 else 0;
    std.debug.print("VAL if_true_inc: {d}\n",.{if_true_inc});

    var i:u32 = 0;
    var sum = i;
    
    while(i <= 100) : (i+=3){
        sum += if (i%2==0) i else continue;
        std.debug.print("VAL sum:{d}\n",.{sum}); 
    }

    const string = [_]u8{'h','l','l','f'};
    
    for (string,0..) |char,index|{
        std.debug.print("VAL i:{d}\n",.{index}); 
        std.debug.print("VAL character:{c}\n",.{char}); 
    }
    
    const y = 10;
    std.debug.print("VAL add_five(y):{d}\n",.{addFive(y)});
    std.debug.print("Val fibonacci(y):{d}\n",.{fibonacci(y)});
}
