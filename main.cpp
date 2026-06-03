extern "C" {
  #include "raylib.h"
}

#include <string>
#include <vector>
#include <fstream>

struct Cursor {
  int currentLine;
  int currentCol;
};

const std::string filePath = "./test.txt";

const int fps = 60;
const int screenWidth = 800;
const int screenHeight = 400;
const int gutterWidth = 20;
const int fontSize = 20;
const int spacing = 1;

bool key_repeat(int key, int &timer, int delay, int rate) {
  if (!IsKeyDown(key)) {
    timer = 0;
    return false;
  }

  timer++;

  if (timer == 1) return true;
  if (timer > delay && (timer - delay) % rate == 0) return true;

  return false;
}

int line_length(const std::vector<std::string>& lines, int line) {
  return (int)lines[line].size();
}

void move_left(const std::vector<std::string>& lines, Cursor& cursor) {
  if (cursor.currentCol > 0) {
    cursor.currentCol--;
  } else if (cursor.currentLine > 0) {
    cursor.currentLine--;
    cursor.currentCol = line_length(lines, cursor.currentLine);
  }
}

void move_right(const std::vector<std::string>& lines, Cursor& cursor) {
  if (cursor.currentCol < line_length(lines, cursor.currentLine)) {
    cursor.currentCol++;
  } else if (cursor.currentLine + 1 < (int)lines.size()) {
    cursor.currentLine++;
    cursor.currentCol = 0;
  }
}

void move_up(const std::vector<std::string>& lines, Cursor& cursor, int& desiredCol) {
  if (cursor.currentLine == 0) return;

  if (desiredCol == -1) desiredCol = cursor.currentCol;

  cursor.currentLine--;
  int targetLength = line_length(lines, cursor.currentLine);
  cursor.currentCol = desiredCol < targetLength ? desiredCol : targetLength;
}

void move_down(const std::vector<std::string>& lines, Cursor& cursor, int& desiredCol) {
  if (cursor.currentLine + 1 >= (int)lines.size()) return;

  if (desiredCol == -1) desiredCol = cursor.currentCol;

  cursor.currentLine++;
  int targetLength = line_length(lines, cursor.currentLine);
  cursor.currentCol = desiredCol < targetLength ? desiredCol : targetLength;
}

void insert_char(std::vector<std::string>& lines, Cursor& cursor, char c) {
  std::string& line = lines[cursor.currentLine];
  line.insert(line.begin() + cursor.currentCol, c);
  cursor.currentCol++;
}

void insert_newline(std::vector<std::string>& lines, Cursor& cursor) {
  std::string& line = lines[cursor.currentLine];
  std::string rest = line.substr(cursor.currentCol);

  line.erase(cursor.currentCol);
  lines.insert(lines.begin() + cursor.currentLine + 1, rest);

  cursor.currentLine++;
  cursor.currentCol = 0;
}

void backspace(std::vector<std::string>& lines, Cursor& cursor) {
  std::string& line = lines[cursor.currentLine];

  if (cursor.currentCol > 0) {
    line.erase(cursor.currentCol - 1, 1);
    cursor.currentCol--;
    return;
  }

  if (cursor.currentLine == 0) return;

  int previousLength = line_length(lines, cursor.currentLine - 1);
  lines[cursor.currentLine - 1] += line;
  lines.erase(lines.begin() + cursor.currentLine);

  cursor.currentLine--;
  cursor.currentCol = previousLength;
}

int main() {
  std::vector<std::string> lines = {""};
  Cursor cursor = {0, 0};
  int desiredCol = -1;

  int arrowTimerRight = 0;
  int arrowTimerLeft = 0;
  int bkSpaceTimer = 0;
  int enterTimer = 0;
  int keyDelay = 30;
  int repeatRate = 2;

  InitWindow(screenWidth, screenHeight, "Sumi");
  SetTargetFPS(fps);

  Font font = LoadFont("/tmp/JetBrainsMono.ttf");
  std::ofstream MyFile(filePath);

  while (!WindowShouldClose()) {
    if (key_repeat(KEY_RIGHT, arrowTimerRight, keyDelay, repeatRate)) {
      move_right(lines, cursor);
      desiredCol = -1;
    }

    if (key_repeat(KEY_LEFT, arrowTimerLeft, keyDelay, repeatRate)) {
      move_left(lines, cursor);
      desiredCol = -1;
    }

    if (IsKeyPressed(KEY_UP)) {
      move_up(lines, cursor, desiredCol);
    }

    if (IsKeyPressed(KEY_DOWN)) {
      move_down(lines, cursor, desiredCol);
    }

    if (key_repeat(KEY_BACKSPACE, bkSpaceTimer, keyDelay, repeatRate)) {
      backspace(lines, cursor);
      desiredCol = -1;
    }

    if (key_repeat(KEY_ENTER, enterTimer, keyDelay, repeatRate)) {
      insert_newline(lines, cursor);
      desiredCol = -1;
    }
    
    if ((IsKeyDown(KEY_LEFT_CONTROL) || IsKeyDown(KEY_RIGHT_CONTROL)) && IsKeyPressed(KEY_S)){
      printf("Saving fileee!");
      for (const auto& line : lines){
        MyFile<<line+"\n";
      }  
    }

    char typedChar = GetCharPressed();
    if (typedChar != 0) {
      insert_char(lines, cursor, typedChar);
      desiredCol = -1;
    }

    BeginDrawing();
    ClearBackground(RAYWHITE);

    float penY = 0;
    float cursorX = gutterWidth;
    float cursorY = 0;

    for (int lineIndex = 0; lineIndex < (int)lines.size(); lineIndex++) {
      float penX = gutterWidth;

      DrawTextEx(font, TextFormat("%d", lineIndex + 1), {0, penY}, fontSize, spacing, GRAY);

      for (int col = 0; col < (int)lines[lineIndex].size(); col++) {
        if (lineIndex == cursor.currentLine && col == cursor.currentCol) {
          cursorX = penX;
          cursorY = penY;
        }

        const char* glyph = TextFormat("%c", lines[lineIndex][col]);
        float glyphW = MeasureTextEx(font, glyph, fontSize, spacing).x;

        if (penX + glyphW > screenWidth) {
          penX = gutterWidth;
          penY += fontSize;
        }

        DrawTextEx(font, glyph, {penX, penY}, fontSize, spacing, RED);
        penX += glyphW;
      }

      if (lineIndex == cursor.currentLine && cursor.currentCol == (int)lines[lineIndex].size()) {
        cursorX = penX;
        cursorY = penY;
      }

      penY += fontSize;
    }

    DrawRectangle(cursorX, cursorY, 2, fontSize, GREEN);

    EndDrawing();
  }

  UnloadFont(font);
  CloseWindow();
  return 0;
}
