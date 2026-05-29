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
}
