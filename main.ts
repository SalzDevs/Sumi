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

//TODO: Refactor this part (validate_next_position should only take the cursor)
function validate_next_position(next_x:number, next_y:number,c:cursor_type) :boolean{
  return (next_x>=0 && next_x<=screenWidth-c.width && next_y>=0 && next_y<=screenHeight-c.height);
}

function render_window(){
  const m:model = {cursor: {x:0,y:0,width:10,height:20}, char_buffer: ""};
  const q = new Queue<string>();

  r.InitWindow(screenWidth,screenHeight, "Sumi");
  r.SetTargetFPS(60);

  while(!WindowShouldClose()){
    //TODO: Refactor this part (validate_next_position should only take the cursor)
    if (r.IsKeyDown(r.KEY_RIGHT) && validate_next_position(m.cursor.x+1,m.cursor.y,m.cursor)) m.cursor.x+=1;
    if (r.IsKeyDown(r.KEY_DOWN) && validate_next_position(m.cursor.x,m.cursor.y+1,m.cursor)) m.cursor.y+=1;
    if (r.IsKeyDown(r.KEY_LEFT) && validate_next_position(m.cursor.x-1,m.cursor.y,m.cursor)) m.cursor.x-=1;
    if (r.IsKeyDown(r.KEY_UP) && validate_next_position(m.cursor.x,m.cursor.y-1,m.cursor)) m.cursor.y-=1;
    const rune = r.GetKeyPressed();
    if ((rune!=0) && !(r.IsKeyDown(r.KEY_RIGHT)) && !(r.IsKeyDown(r.KEY_DOWN)) && !(r.IsKeyDown(r.KEY_LEFT)) && !(r.IsKeyDown(r.KEY_UP))){
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
