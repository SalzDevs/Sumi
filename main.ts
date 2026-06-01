import r, { InitWindow, SetTargetFPS, WindowShouldClose } from 'raylib';

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

function render_window(){
  const screenWidth = 800;
  const screenHeight = 400;
  const m:model = {cursor: {x:0,y:0,width:10,height:20}, char_buffer: ""};

  r. InitWindow(screenWidth,screenHeight, "Sumi");
  r.SetTargetFPS(60);

  while(!WindowShouldClose()){
    if (r.IsKeyDown(r.KEY_RIGHT)) m.cursor.x+=1;
    if (r.IsKeyDown(r.KEY_DOWN)) m.cursor.y+=1;
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
