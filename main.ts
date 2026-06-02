import r, { InitWindow, SetTargetFPS, WindowShouldClose } from 'raylib';

const screenWidth = 800;
const screenHeight = 400;

const fontSize = 20;

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

interface rune{
  char:string
  x:number
  y:number
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

  r.InitWindow(screenWidth,screenHeight, "Sumi");
  r.SetTargetFPS(60);
  
  const runes :rune[] = [];

  while(!WindowShouldClose()){
    try_move_if(r.KEY_RIGHT,m.cursor,1,0);
    try_move_if(r.KEY_DOWN,m.cursor,0,1);
    try_move_if(r.KEY_LEFT,m.cursor,-1,0);
    try_move_if(r.KEY_UP,m.cursor,0,-1);

    const c = r.GetKeyPressed();
    if (c != 0){
      runes.push({char: String.fromCharCode(c), x: m.cursor.x+5, y: m.cursor.y});
      if (m.cursor.x + 6 < screenWidth - m.cursor.width) m.cursor.x += 6;
      else{
        if (m.cursor.y + 1 < screenHeight - m.cursor.height){ 
          m.cursor.y += fontSize;
          m.cursor.x = 0;
        }  
      }
    }

    r.BeginDrawing();
    for (const elem of runes){
      r.DrawText(elem.char,elem.x,elem.y,fontSize,r.RED);
    }
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
