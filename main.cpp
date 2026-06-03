extern "C" {
  #include "raylib.h"
}
#include <string>

const int fps = 60;
const int screenWidth = 800;
const int screenHeight = 400;


const int fontSize = 20;
const int spacing = 1;

bool key_repeat(int key,int &timer,int delay,int rate){
  if (!IsKeyDown(key)){
    timer = 0;
    return false;
  }

  timer++;
  
  if (timer==1) return true;
  
  if (timer>delay && (timer-delay)%rate==0) return true;
  
  return false;
}

int main() {
  std::string text;
  int cursorIndex = 0;

  int arrowTimerRight = 0;
  int arrowTimerLeft = 0;
  int bkSpaceTimer = 0;
  int enterTimer = 0;
  int key_delay = 30;
  int rate = 2;
  
  InitWindow(screenWidth, screenHeight, "Sumi");
  SetTargetFPS(fps);
  
  Font font = LoadFont("/tmp/JetBrainsMono.ttf");
  
  while (!WindowShouldClose()) {
    if (key_repeat(KEY_RIGHT,arrowTimerRight,key_delay,rate) && cursorIndex < (int)text.size()) cursorIndex ++;
    if (key_repeat(KEY_LEFT,arrowTimerLeft,key_delay,rate)  && cursorIndex > 0) cursorIndex--;

    if (key_repeat(KEY_BACKSPACE,bkSpaceTimer,key_delay,rate) && cursorIndex > 0) {
      text.erase(cursorIndex - 1, 1);
      cursorIndex--;
    }
    
    if (key_repeat(KEY_ENTER,enterTimer,key_delay,rate)) {
      text.insert(text.begin() + cursorIndex, '\n');
      cursorIndex++;
    }

    char c = GetCharPressed();
    if (c != 0) {
      text.insert(text.begin() + cursorIndex, c);
      cursorIndex++;
    }


    BeginDrawing();
    ClearBackground(RAYWHITE);

    float penX = 20;
    float penY = 0;
    float cursorX = 20, cursorY = 0;
    int lineNum = 1;

    DrawTextEx(font, TextFormat("%d", lineNum), {0, penY}, fontSize, spacing, GRAY);

    for (int i = 0; i < (int)text.size(); i++) {
      if (i == cursorIndex) {
        cursorX = penX;
        cursorY = penY;
      }
      
      if (text[i] == '\n') {
        penX = 20;
        penY += fontSize;
        lineNum++;
        DrawTextEx(font, TextFormat("%d", lineNum), {0, penY}, fontSize, spacing, GRAY);
        continue;
      }

      const char* glyph = TextFormat("%c", text[i]);
      float glyphW = MeasureTextEx(font, glyph, fontSize, spacing).x;

      if (penX + glyphW > screenWidth) {
        penX = 20;
        penY += fontSize;
        lineNum++;
        DrawTextEx(font, TextFormat("%d", lineNum), {0, penY}, fontSize, spacing, GRAY);
      }

      DrawTextEx(font, glyph, {penX, penY}, fontSize, spacing, RED);
      penX += glyphW;
    }

    if (cursorIndex == (int)text.size()) {
      cursorX = penX;
      cursorY = penY;
    }

    DrawRectangle(cursorX, cursorY, 2, fontSize, GREEN);
    
    EndDrawing();
  }

  UnloadFont(font);
  CloseWindow();
  return 0;
}
