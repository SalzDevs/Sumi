import r, { InitWindow, SetTargetFPS, WindowShouldClose } from 'raylib';

function main(){
  const screenWidth = 800;
  const screenHeight = 400;

 r. InitWindow(screenWidth,screenHeight, "Sumi");

  r.SetTargetFPS(60);

  while(!WindowShouldClose()){
    r.BeginDrawing();
    r.ClearBackground(r.RAYWHITE);
    r.DrawText("Congrats! You created your first window!", 190, 200, 20, r.LIGHTGRAY);
    r.EndDrawing();
  }
  r.CloseWindow();
}

main();
