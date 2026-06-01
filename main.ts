import r, { InitWindow, SetTargetFPS, WindowShouldClose } from 'raylib';

interface model{
  cursor:number
  char_buffer:string 
}

function render_window(){
  const screenWidth = 800;
  const screenHeight = 400;

  r. InitWindow(screenWidth,screenHeight, "Sumi");
  r.SetTargetFPS(60);

  while(!WindowShouldClose()){
    r.BeginDrawing();
    r.ClearBackground(r.RAYWHITE);
    r.DrawRectangle(0,0,10,20,r.GREEN);
    r.EndDrawing();
  }
  r.CloseWindow();
}

function main(){
  const m:model = {cursor: 0, char_buffer: ""};
  render_window();
}

main();
