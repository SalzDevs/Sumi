extern "C" {
#include "raylib.h"
}

#include <fstream>
#include <string>
#include <vector>
#include <unordered_map>


int keyDelay = 30;
int repeatRate = 2;

struct Buffer {
  std::vector<std::string> lines = {""};
  std::string filePath = "./test.txt";
  bool modified = false;
};

struct Cursor {
  int currentLine = 0;
  int currentCol = 0;
  int desiredCol = -1;
};

struct View {
  int gutterWidth = 20;
  int fontSize = 20;
  int spacing = 1;
};

struct Editor {
  Buffer buffer;
  Cursor cursor;
  View view;
};

struct Command {
  std::string name;
  void(*action)(Editor&);
};

struct CommandRegistry {
  std::unordered_map<std::string, Command> commands;
  std::unordered_map<int, std::string> keyMap;
  std::unordered_map<int, int> timers;

  void register_command(const std::string& name, void(*action)(Editor&)) {
    commands[name] = {name, action};
  }

  void bind_key(int key, const std::string& name) {
    keyMap[key] = name;
    timers[key] = 0;
  }

  void execute(const std::string& name, Editor& editor) {
    auto it = commands.find(name);
    if (it != commands.end() && it->second.action) {
      it->second.action(editor);
    }
  }
};

const int fps = 60;
const int screenWidth = 800;
const int screenHeight = 400;

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

int line_length(const Buffer& buffer, int line) {
  return (int)buffer.lines[line].size();
}

void reset_desired_col(Editor& editor) {
  editor.cursor.desiredCol = -1;
}

void move_left(Editor& editor) {
  Buffer& buffer = editor.buffer;
  Cursor& cursor = editor.cursor;

  if (cursor.currentCol > 0) {
    cursor.currentCol--;
  } else if (cursor.currentLine > 0) {
    cursor.currentLine--;
    cursor.currentCol = line_length(buffer, cursor.currentLine);
  }

  reset_desired_col(editor);
}

void move_right(Editor& editor) {
  Buffer& buffer = editor.buffer;
  Cursor& cursor = editor.cursor;

  if (cursor.currentCol < line_length(buffer, cursor.currentLine)) {
    cursor.currentCol++;
  } else if (cursor.currentLine + 1 < (int)buffer.lines.size()) {
    cursor.currentLine++;
    cursor.currentCol = 0;
  }

  reset_desired_col(editor);
}

void move_up(Editor& editor) {
  Buffer& buffer = editor.buffer;
  Cursor& cursor = editor.cursor;

  if (cursor.currentLine == 0) return;

  if (cursor.desiredCol == -1) cursor.desiredCol = cursor.currentCol;

  cursor.currentLine--;
  int targetLength = line_length(buffer, cursor.currentLine);
  cursor.currentCol = cursor.desiredCol < targetLength ? cursor.desiredCol : targetLength;
}

void move_down(Editor& editor) {
  Buffer& buffer = editor.buffer;
  Cursor& cursor = editor.cursor;

  if (cursor.currentLine + 1 >= (int)buffer.lines.size()) return;

  if (cursor.desiredCol == -1) cursor.desiredCol = cursor.currentCol;

  cursor.currentLine++;
  int targetLength = line_length(buffer, cursor.currentLine);
  cursor.currentCol = cursor.desiredCol < targetLength ? cursor.desiredCol : targetLength;
}

void insert_char(Editor& editor, char c) {
  Buffer& buffer = editor.buffer;
  Cursor& cursor = editor.cursor;

  std::string& line = buffer.lines[cursor.currentLine];
  line.insert(line.begin() + cursor.currentCol, c);
  cursor.currentCol++;

  buffer.modified = true;
  reset_desired_col(editor);
}

void insert_newline(Editor& editor) {
  Buffer& buffer = editor.buffer;
  Cursor& cursor = editor.cursor;

  std::string& line = buffer.lines[cursor.currentLine];
  std::string rest = line.substr(cursor.currentCol);

  line.erase(cursor.currentCol);
  buffer.lines.insert(buffer.lines.begin() + cursor.currentLine + 1, rest);

  cursor.currentLine++;
  cursor.currentCol = 0;

  buffer.modified = true;
  reset_desired_col(editor);
}

void backspace(Editor& editor) {
  Buffer& buffer = editor.buffer;
  Cursor& cursor = editor.cursor;
  std::string& line = buffer.lines[cursor.currentLine];

  if (cursor.currentCol > 0) {
    line.erase(cursor.currentCol - 1, 1);
    cursor.currentCol--;
    buffer.modified = true;
    reset_desired_col(editor);
    return;
  }

  if (cursor.currentLine == 0) return;

  int previousLength = line_length(buffer, cursor.currentLine - 1);
  buffer.lines[cursor.currentLine - 1] += line;
  buffer.lines.erase(buffer.lines.begin() + cursor.currentLine);

  cursor.currentLine--;
  cursor.currentCol = previousLength;

  buffer.modified = true;
  reset_desired_col(editor);
}

void save_file(Editor& editor) {
  std::ofstream file(editor.buffer.filePath);

  if (!file) {
    printf("Failed to save file!\n");
    return;
  }

  for (const auto& line : editor.buffer.lines) {
    file << line << '\n';
  }

  editor.buffer.modified = false;
}

void handle_input(Editor& editor, CommandRegistry& registry) {
  for (const auto& [key, cmd_name] : registry.keyMap) {
    bool triggered = false;
    if (key == KEY_UP || key == KEY_DOWN) {
      triggered = IsKeyPressed(key);
    } else {
      int& timer = registry.timers[key];
      triggered = key_repeat(key, timer, keyDelay, repeatRate);
    }
    if (triggered) {
      registry.execute(cmd_name, editor);
    }
  }

  if ((IsKeyDown(KEY_LEFT_CONTROL) || IsKeyDown(KEY_RIGHT_CONTROL)) &&
      IsKeyPressed(KEY_S)) {
    registry.execute("save_file", editor);
  }
}

void render_editor(Editor& editor, Font font) {
  float penY = 0;
  float cursorX = editor.view.gutterWidth;
  float cursorY = 0;

  for (int lineIndex = 0; lineIndex < (int)editor.buffer.lines.size(); lineIndex++) {
    float penX = editor.view.gutterWidth;

    DrawTextEx(font, TextFormat("%d", lineIndex + 1), {0, penY}, editor.view.fontSize, editor.view.spacing, GRAY);

    for (int col = 0; col < (int)editor.buffer.lines[lineIndex].size(); col++) {
      if (lineIndex == editor.cursor.currentLine && col == editor.cursor.currentCol) {
        cursorX = penX;
        cursorY = penY;
      }

      const char* glyph = TextFormat("%c", editor.buffer.lines[lineIndex][col]);
      float glyphW = MeasureTextEx(font, glyph, editor.view.fontSize, editor.view.spacing).x;

      if (penX + glyphW > screenWidth) {
        penX = editor.view.gutterWidth;
        penY += editor.view.fontSize;
      }

      DrawTextEx(font, glyph, {penX, penY}, editor.view.fontSize, editor.view.spacing, RED);
      penX += glyphW;
    }

    if (lineIndex == editor.cursor.currentLine && editor.cursor.currentCol == (int)editor.buffer.lines[lineIndex].size()) {
      cursorX = penX;
      cursorY = penY;
    }

    penY += editor.view.fontSize;
  }

  DrawRectangle(cursorX, cursorY, 2, editor.view.fontSize, GREEN);
}

int main() {
  Editor editor;
  CommandRegistry registry;

  registry.register_command("move_right", move_right);
  registry.register_command("move_left", move_left);
  registry.register_command("move_up", move_up);
  registry.register_command("move_down", move_down);
  registry.register_command("backspace", backspace);
  registry.register_command("insert_newline", insert_newline);
  registry.register_command("save_file", save_file);

  registry.bind_key(KEY_RIGHT, "move_right");
  registry.bind_key(KEY_LEFT, "move_left");
  registry.bind_key(KEY_UP, "move_up");
  registry.bind_key(KEY_DOWN, "move_down");
  registry.bind_key(KEY_BACKSPACE, "backspace");
  registry.bind_key(KEY_ENTER, "insert_newline");

  InitWindow(screenWidth, screenHeight, "Sumi");
  SetTargetFPS(fps);

  Font font = LoadFont("assets/JetBrainsMono-Regular.ttf");
  bool customFontLoaded = font.texture.id != 0;
  if (!customFontLoaded) font = GetFontDefault();

  while (!WindowShouldClose()) {
    handle_input(editor, registry);

    char typedChar = GetCharPressed();
    if (typedChar != 0) {
      insert_char(editor, typedChar);
    }

    BeginDrawing();
    ClearBackground(RAYWHITE);
    render_editor(editor, font);
    EndDrawing();
  }

  if (customFontLoaded) UnloadFont(font);
  CloseWindow();
  return 0;
}
