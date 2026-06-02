extern "C" {
#include "raylib.h"
}
#include <vector>

const int screenWidth = 800;
const int screenHeight = 400;
const int fontSize = 20;
const int spacing = 2;
struct Cursor {
  int x, y;
  int width, height;
};

struct Rune {
  char c;
  int x, y;
};

bool in_bounds(const Cursor& cursor) {
  return (cursor.x >= 0 &&
          cursor.x <= screenWidth - cursor.width &&
          cursor.y >= 0 &&
          cursor.y <= screenHeight - cursor.height);
}

void try_move_if(int key, Cursor& cursor, int dx, int dy) {
  if (!IsKeyDown(key)) return;
  cursor.x += dx;
  cursor.y += dy;
  if (!in_bounds(cursor)) {
    cursor.x -= dx;
    cursor.y -= dy;
  }
}

int main() {
  Cursor cursor = {0, 0, 10, 20};
  std::vector<Rune> runes;

  InitWindow(screenWidth, screenHeight, "Sumi");
  SetTargetFPS(60);
  
  Font font = LoadFont("/tmp/JetBrainsMono.ttf");
  
  while (!WindowShouldClose()) {
    try_move_if(KEY_RIGHT, cursor, 1, 0);
    try_move_if(KEY_DOWN,  cursor, 0, 1);
    try_move_if(KEY_LEFT,  cursor, -1, 0);
    try_move_if(KEY_UP,    cursor, 0, -1);

    int c = GetKeyPressed();
    if (c != 0) {
      runes.push_back({(char)c, cursor.x, cursor.y});
      Vector2 glyphSize = MeasureTextEx(font,TextFormat("%c",c),fontSize,spacing);
      if (cursor.x + 1 < screenWidth - cursor.width) {
        cursor.x += glyphSize.x;
      } else {
        if (cursor.y + fontSize < screenHeight - cursor.height) {
          cursor.y += fontSize;
          cursor.x = 0;
        }
      }
    }

    BeginDrawing();
    ClearBackground(RAYWHITE);

    for (auto& rn : runes) {
      DrawTextEx(font,TextFormat("%c", rn.c), {static_cast<float>(rn.x), static_cast<float>(rn.y)}, fontSize,4, RED);
    }

    DrawRectangle(cursor.x, cursor.y, cursor.width, cursor.height, GREEN);
    EndDrawing();
  }
  UnloadFont(font);
  CloseWindow();
  return 0;
}
