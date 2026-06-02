extern "C" {
  #include "raylib.h"
}
#include <string>

const int screenWidth = 800;
const int screenHeight = 400;
const int fontSize = 20;
const int spacing = 1;

int main() {
  std::string text;
  int cursorIndex = 0;

  InitWindow(screenWidth, screenHeight, "Sumi");
  SetTargetFPS(60);
  
  Font font = LoadFont("/tmp/JetBrainsMono.ttf");

  while (!WindowShouldClose()) {

    if (IsKeyPressed(KEY_RIGHT) && cursorIndex < (int)text.size()) cursorIndex++;
    if (IsKeyPressed(KEY_LEFT)  && cursorIndex > 0)                  cursorIndex--;

    if (IsKeyPressed(KEY_BACKSPACE) && cursorIndex > 0) {
      text.erase(cursorIndex - 1, 1);
      cursorIndex--;
    }

    char c = GetCharPressed();
    if (c != 0) {
      text.insert(text.begin() + cursorIndex, c);
      cursorIndex++;
    }


    BeginDrawing();
    ClearBackground(RAYWHITE);

    float penX = 0;
    float penY = 0;
    float cursorX = 0, cursorY = 0;

    for (int i = 0; i < (int)text.size(); i++) {
      if (i == cursorIndex) {
        cursorX = penX;
        cursorY = penY;
      }

      const char* glyph = TextFormat("%c", text[i]);
      float glyphW = MeasureTextEx(font, glyph, fontSize, spacing).x;

      if (penX + glyphW > screenWidth) {
        penX = 0;
        penY += fontSize;
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
