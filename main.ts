import r, { InitWindow, SetTargetFPS, WindowShouldClose } from 'raylib';
import {Queue} from '@datastructures-js/queue';

const screenWidth = 800;
const screenHeight = 400;


interface cursor_type{
  x:number
  y:number
  width:number
  height:number
}

interface model{
  cursor:cursor_type
  char_buffer:string 
}

function in_bounds(cursor: cursor_type) :boolean{
  return ((cursor.x >= 0) && (cursor.x <= screenWidth-cursor.width) && (cursor.y >= 0) && (cursor.y <= screenHeight-cursor.height));
}

function try_move_if(key:number,cursor:cursor_type,dx:number,dy:number){
  if (!r.IsKeyDown(key)) return; 
  cursor.x +=dx;
  cursor.y +=dy;
  if(!in_bounds(cursor)){
    cursor.x-=dx,
    cursor.y-=dy
    return 
  }
}

function render_window(){
  const m:model = {cursor: {x:0,y:0,width:10,height:20}, char_buffer: ""};
  const q = new Queue<string>();

  r.InitWindow(screenWidth,screenHeight, "Sumi");
  r.SetTargetFPS(60);

  while(!WindowShouldClose()){
    try_move_if(r.KEY_RIGHT,m.cursor,1,0);
    try_move_if(r.KEY_DOWN,m.cursor,0,1);
    try_move_if(r.KEY_LEFT,m.cursor,-1,0);
    try_move_if(r.KEY_UP,m.cursor,0,-1);

    const rune = r.GetKeyPressed();
    if (rune!=0){
       q.push(String.fromCharCode(rune)); 
    }

    r.BeginDrawing();
    r.ClearBackground(r.RAYWHITE);
    r.DrawRectangle(m.cursor.x,m.cursor.y,m.cursor.width,m.cursor.height,r.GREEN);
    r.EndDrawing();
  }
  console.log(q)
  r.CloseWindow();
}

function main(){
  render_window();
}

main();
