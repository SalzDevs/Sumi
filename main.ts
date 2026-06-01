import r, { InitWindow, SetTargetFPS, WindowShouldClose } from 'raylib';

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

function validate_next_position(next_x:number, next_y:number,c:cursor_type) :boolean{
  if (next_x<0 || next_x>screenWidth-c.width || next_y<0||next_y>screenHeight-c.height) return false;
  console.log("x:  y:",next_x,next_y);
  return true; 
}

function render_window(){
  const m:model = {cursor: {x:0,y:0,width:10,height:20}, char_buffer: ""};

  r. InitWindow(screenWidth,screenHeight, "Sumi");
  r.SetTargetFPS(60);

  while(!WindowShouldClose()){
    if (r.IsKeyDown(r.KEY_RIGHT) && validate_next_position(m.cursor.x+1,m.cursor.y,m.cursor)) m.cursor.x+=1;
    if (r.IsKeyDown(r.KEY_DOWN) && validate_next_position(m.cursor.x,m.cursor.y+1,m.cursor)) m.cursor.y+=1;
    if (r.IsKeyDown(r.KEY_LEFT) && validate_next_position(m.cursor.x-1,m.cursor.y,m.cursor)) m.cursor.x-=1;
    if (r.IsKeyDown(r.KEY_UP) && validate_next_position(m.cursor.x,m.cursor.y-1,m.cursor)) m.cursor.y-=1;

    r.BeginDrawing();
    r.ClearBackground(r.RAYWHITE);
    r.DrawRectangle(m.cursor.x,m.cursor.y,m.cursor.width,m.cursor.height,r.GREEN);
    r.EndDrawing();
  }
  r.CloseWindow();
}

function main(){
  render_window();
}

main();
