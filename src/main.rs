use raylib::prelude::*;
use std::collections::HashMap;
use std::fs::File;
use std::io::Write;

const FPS: u32 = 60;
const SCREEN_WIDTH: i32 = 800;
const SCREEN_HEIGHT: i32 = 400;
const KEY_DELAY: i32 = 30;
const REPEAT_RATE: i32 = 2;

struct Buffer {
    lines: Vec<String>,
    file_path: String,
    modified: bool,
}

impl Default for Buffer {
    fn default() -> Self {
        Self {
            lines: vec![String::new()],
            file_path: String::from("./test.txt"),
            modified: false,
        }
    }
}

struct Cursor {
    current_line: usize,
    current_col: usize,
    desired_col: Option<usize>,
}

impl Default for Cursor {
    fn default() -> Self {
        Self {
            current_line: 0,
            current_col: 0,
            desired_col: None,
        }
    }
}

struct View {
    gutter_width: i32,
    font_size: i32,
    spacing: i32,
}

impl Default for View {
    fn default() -> Self {
        Self {
            gutter_width: 20,
            font_size: 20,
            spacing: 1,
        }
    }
}

struct Editor {
    buffer: Buffer,
    cursor: Cursor,
    view: View,
}

impl Default for Editor {
    fn default() -> Self {
        Self {
            buffer: Buffer::default(),
            cursor: Cursor::default(),
            view: View::default(),
        }
    }
}

struct Command {
    #[allow(dead_code)]
    name: String,
    action: fn(&mut Editor),
}

struct CommandRegistry {
    commands: HashMap<String, Command>,
    key_map: HashMap<KeyboardKey, String>,
    timers: HashMap<KeyboardKey, i32>,
}

impl CommandRegistry {
    fn new() -> Self {
        Self {
            commands: HashMap::new(),
            key_map: HashMap::new(),
            timers: HashMap::new(),
        }
    }

    fn register_command(&mut self, name: &str, action: fn(&mut Editor)) {
        self.commands.insert(
            name.to_string(),
            Command {
                name: name.to_string(),
                action,
            },
        );
    }

    fn bind_key(&mut self, key: KeyboardKey, name: &str) {
        self.key_map.insert(key, name.to_string());
        self.timers.insert(key, 0);
    }

    fn execute(&self, name: &str, editor: &mut Editor) {
        if let Some(cmd) = self.commands.get(name) {
            (cmd.action)(editor);
        }
    }
}

fn key_repeat(rl: &RaylibHandle, key: KeyboardKey, timer: &mut i32, delay: i32, rate: i32) -> bool {
    if !rl.is_key_down(key) {
        *timer = 0;
        return false;
    }

    *timer += 1;

    if *timer == 1 {
        return true;
    }
    if *timer > delay && (*timer - delay) % rate == 0 {
        return true;
    }

    false
}

fn line_length(buffer: &Buffer, line: usize) -> usize {
    buffer.lines[line].len()
}

fn reset_desired_col(editor: &mut Editor) {
    editor.cursor.desired_col = None;
}

fn move_left(editor: &mut Editor) {
    let buffer = &editor.buffer;
    let cursor = &mut editor.cursor;

    if cursor.current_col > 0 {
        cursor.current_col -= 1;
    } else if cursor.current_line > 0 {
        cursor.current_line -= 1;
        cursor.current_col = line_length(buffer, cursor.current_line);
    }

    reset_desired_col(editor);
}

fn move_right(editor: &mut Editor) {
    let buffer = &editor.buffer;
    let cursor = &mut editor.cursor;

    if cursor.current_col < line_length(buffer, cursor.current_line) {
        cursor.current_col += 1;
    } else if cursor.current_line + 1 < buffer.lines.len() {
        cursor.current_line += 1;
        cursor.current_col = 0;
    }

    reset_desired_col(editor);
}

fn move_up(editor: &mut Editor) {
    let buffer = &editor.buffer;
    let cursor = &mut editor.cursor;

    if cursor.current_line == 0 {
        return;
    }

    if cursor.desired_col.is_none() {
        cursor.desired_col = Some(cursor.current_col);
    }

    cursor.current_line -= 1;
    let target_length = line_length(buffer, cursor.current_line);
    let desired = cursor.desired_col.unwrap_or(0);
    cursor.current_col = desired.min(target_length);
}

fn move_down(editor: &mut Editor) {
    let buffer = &editor.buffer;
    let cursor = &mut editor.cursor;

    if cursor.current_line + 1 >= buffer.lines.len() {
        return;
    }

    if cursor.desired_col.is_none() {
        cursor.desired_col = Some(cursor.current_col);
    }

    cursor.current_line += 1;
    let target_length = line_length(buffer, cursor.current_line);
    let desired = cursor.desired_col.unwrap_or(0);
    cursor.current_col = desired.min(target_length);
}

fn insert_char(editor: &mut Editor, c: char) {
    let buffer = &mut editor.buffer;
    let cursor = &mut editor.cursor;

    let line = &mut buffer.lines[cursor.current_line];
    line.insert(cursor.current_col, c);
    cursor.current_col += 1;

    buffer.modified = true;
    reset_desired_col(editor);
}

fn insert_newline(editor: &mut Editor) {
    let buffer = &mut editor.buffer;
    let cursor = &mut editor.cursor;

    let line = &mut buffer.lines[cursor.current_line];
    let rest: String = line.split_off(cursor.current_col);

    buffer.lines.insert(cursor.current_line + 1, rest);

    cursor.current_line += 1;
    cursor.current_col = 0;

    buffer.modified = true;
    reset_desired_col(editor);
}

fn backspace(editor: &mut Editor) {
    let buffer = &mut editor.buffer;
    let cursor = &mut editor.cursor;
    let line = &mut buffer.lines[cursor.current_line];

    if cursor.current_col > 0 {
        line.remove(cursor.current_col - 1);
        cursor.current_col -= 1;
        buffer.modified = true;
        reset_desired_col(editor);
        return;
    }

    if cursor.current_line == 0 {
        return;
    }

    let previous_length = line_length(buffer, cursor.current_line - 1);
    let removed_line = buffer.lines.remove(cursor.current_line);
    buffer.lines[cursor.current_line - 1].push_str(&removed_line);

    cursor.current_line -= 1;
    cursor.current_col = previous_length;

    buffer.modified = true;
    reset_desired_col(editor);
}

fn save_file(editor: &mut Editor) {
    let mut file = match File::create(&editor.buffer.file_path) {
        Ok(f) => f,
        Err(_) => {
            eprintln!("Failed to save file!");
            return;
        }
    };

    for line in &editor.buffer.lines {
        if let Err(_) = writeln!(file, "{}", line) {
            eprintln!("Failed to write line!");
            return;
        }
    }

    editor.buffer.modified = false;
}

fn handle_input(rl: &mut RaylibHandle, editor: &mut Editor, registry: &mut CommandRegistry) {
    for (&key, cmd_name) in &registry.key_map {
        let triggered = if key == KeyboardKey::KEY_UP || key == KeyboardKey::KEY_DOWN {
            rl.is_key_pressed(key)
        } else {
            let timer = registry.timers.get_mut(&key).unwrap();
            key_repeat(rl, key, timer, KEY_DELAY, REPEAT_RATE)
        };
        if triggered {
            registry.execute(cmd_name, editor);
        }
    }

    if (rl.is_key_down(KeyboardKey::KEY_LEFT_CONTROL)
        || rl.is_key_down(KeyboardKey::KEY_RIGHT_CONTROL))
        && rl.is_key_pressed(KeyboardKey::KEY_S)
    {
        registry.execute("save_file", editor);
    }

    if let Some(ch) = rl.get_char_pressed() {
        insert_char(editor, ch);
    }
}

fn render_editor(d: &mut RaylibDrawHandle, editor: &Editor, font: &Font) {
    let mut pen_y = 0.0f32;
    let mut cursor_x = editor.view.gutter_width as f32;
    let mut cursor_y = 0.0f32;

    for (line_idx, line) in editor.buffer.lines.iter().enumerate() {
        let mut pen_x = editor.view.gutter_width as f32;

        d.draw_text_ex(
            font,
            &format!("{}", line_idx + 1),
            Vector2::new(0.0, pen_y),
            editor.view.font_size as f32,
            editor.view.spacing as f32,
            Color::GRAY,
        );

        for (col, ch) in line.chars().enumerate() {
            if line_idx == editor.cursor.current_line && col == editor.cursor.current_col {
                cursor_x = pen_x;
                cursor_y = pen_y;
            }

            let ch_str = ch.to_string();
            let glyph_w = font.measure_text(
                &ch_str,
                editor.view.font_size as f32,
                editor.view.spacing as f32,
            )
            .x;

            if pen_x + glyph_w > SCREEN_WIDTH as f32 {
                pen_x = editor.view.gutter_width as f32;
                pen_y += editor.view.font_size as f32;
            }

            d.draw_text_ex(
                font,
                &ch_str,
                Vector2::new(pen_x, pen_y),
                editor.view.font_size as f32,
                editor.view.spacing as f32,
                Color::RED,
            );

            pen_x += glyph_w;
        }

        if line_idx == editor.cursor.current_line && editor.cursor.current_col == line.len() {
            cursor_x = pen_x;
            cursor_y = pen_y;
        }

        pen_y += editor.view.font_size as f32;
    }

    d.draw_rectangle(
        cursor_x as i32,
        cursor_y as i32,
        2,
        editor.view.font_size,
        Color::GREEN,
    );
}

fn main() {
    let mut editor = Editor::default();
    let mut registry = CommandRegistry::new();

    registry.register_command("move_right", move_right);
    registry.register_command("move_left", move_left);
    registry.register_command("move_up", move_up);
    registry.register_command("move_down", move_down);
    registry.register_command("backspace", backspace);
    registry.register_command("insert_newline", insert_newline);
    registry.register_command("save_file", save_file);

    registry.bind_key(KeyboardKey::KEY_RIGHT, "move_right");
    registry.bind_key(KeyboardKey::KEY_LEFT, "move_left");
    registry.bind_key(KeyboardKey::KEY_UP, "move_up");
    registry.bind_key(KeyboardKey::KEY_DOWN, "move_down");
    registry.bind_key(KeyboardKey::KEY_BACKSPACE, "backspace");
    registry.bind_key(KeyboardKey::KEY_ENTER, "insert_newline");

    let (mut rl, thread) = raylib::init()
        .size(SCREEN_WIDTH, SCREEN_HEIGHT)
        .title("Sumi")
        .build();

    rl.set_target_fps(FPS);

    let font = rl
        .load_font(&thread, "assets/JetBrainsMono-Regular.ttf")
        .unwrap_or_else(|_| {
            let weak = rl.get_font_default();
            unsafe { Font::from_raw(*weak.as_ref()) }
        });

    while !rl.window_should_close() {
        handle_input(&mut rl, &mut editor, &mut registry);

        let mut d = rl.begin_drawing(&thread);
        d.clear_background(Color::RAYWHITE);
        render_editor(&mut d, &editor, &font);
    }
}
