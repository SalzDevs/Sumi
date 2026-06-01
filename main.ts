import r, { InitWindow, SetTargetFPS, WindowShouldClose } from 'raylib';

interface cursor_type{
  x:number
  y:number
}

interface model{
  cursor:cursor_type
  char_buffer:string 
}

function render_window(){
  const screenWidth = 800;
  const screenHeight = 400;
  const m:model = {cursor: {x:0,y:0}, char_buffer: ""};

  r. InitWindow(screenWidth,screenHeight, "Sumi");
  r.SetTargetFPS(60);

  while(!WindowShouldClose()){
    r.BeginDrawing();
    r.ClearBackground(r.RAYWHITE);
    r.DrawRectangle(m.cursor.x,m.cursor.y,10,20,r.GREEN);
    r.EndDrawing();
  }
  r.CloseWindow();
}

function main(){
  render_window();
}

main();
