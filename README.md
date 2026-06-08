# Sumi

A minimal text editor with a Lua SDK. Ship it bare, configure 99% in `init.lua`.

## Build

Requires Go 1.23+ and raylib (installed automatically via Go modules).

```bash
git clone https://github.com/SalzDevs/Sumi.git
cd Sumi
go build .
```

## Run

```bash
./sumi                    # open ./test.txt
./sumi path/to/file.txt   # open specific file
```

## Configuration

On startup, Sumi loads `~/.config/sumi/init.lua`. If it doesn't exist, the embedded `default.lua` is used.

Create your configuration:

```bash
mkdir -p ~/.config/sumi
cat > ~/.config/sumi/init.lua << 'EOF'
-- Rebind arrow keys to inverted
keymap:Register("normal", keys.LEFT, "move_right")
keymap:Register("normal", keys.RIGHT, "move_left")

-- Register a custom command
commands:Register("uppercase_line", "Uppercase current line", 0, 0, function(e, args)
    local n = e.Cursor:Line()
    e.Buffer:SetLine(n, string.upper(e.Buffer:GetLine(n)))
end)

keymap:Register("normal", keys.HOME, "uppercase_line")
EOF
```

Restart Sumi. No recompile.

## Lua API

### `editor`

| Method | Description |
|---|---|
| `editor:Mode()` | Returns `"normal"`, `"command"`, or `"visual"` |
| `editor:SetMode(mode)` | Set mode string |
| `editor:LineCount()` | Number of lines in buffer |
| `editor:Modified()` | Boolean |
| `editor:LoadFile(path)` | Load file; returns error string or nil |
| `editor:SaveFile()` | Save file; returns error string or nil |
| `editor:Undo()` | Undo last change |
| `editor:Quit()` | Set quit flag |
| `editor:EnterVisual()` | Enter visual mode at cursor |
| `editor:ClearVisual()` | Cancel visual mode |
| `editor:SetVisualAnchor()` | Set anchor to cursor position |
| `editor:SelectWordAt(line, col)` | Select word at 1-based position |
| `editor:SelectLineAt(line)` | Select entire 1-based line |
| `editor:Backspace()` | Delete character before cursor |
| `editor:InsertNewline()` | Insert newline |
| `editor:InsertChar(ch)` | Insert single character string |
| `editor:Yank()` | Copy selection to clipboard; returns error or nil |
| `editor:Paste()` | Paste clipboard at cursor |
| `editor:DeleteSelection()` | Delete visual selection |
| `editor:CommandLine()` | Get current command line string |
| `editor:SetCommandLine(text)` | Set command line string |
| `editor:CommandLineBackspace()` | Delete last command character |

### `editor.Buffer`

| Method | Description |
|---|---|
| `Buffer:FilePath()` | Path of the current file, or empty string |
| `Buffer:GetLine(n)` | Get 1-based line as string |
| `Buffer:SetLine(n, text)` | Replace 1-based line |
| `Buffer:LineCount()` | Number of lines |
| `Buffer:InsertChar(line, col, ch)` | Insert char at 1-based position |
| `Buffer:DeleteChar(line, col)` | Delete char at 1-based position |

### `editor.Cursor`

| Method | Description |
|---|---|
| `Cursor:Line()` | 1-based line |
| `Cursor:Col()` | 1-based column |
| `Cursor:Goto(line, col)` | Move cursor; clamps to valid bounds |
| `Cursor:MoveLeft()` | |
| `Cursor:MoveRight()` | |
| `Cursor:MoveUp()` | |
| `Cursor:MoveDown()` | |

### `commands`

| Method | Description |
|---|---|
| `commands:Register(name, desc, minArgs, maxArgs, handler)` | Register a command. Handler receives `(editor, args[])` and returns `nil` or error string. |
| `commands:List()` | Returns array of registered command names |

### `keymap`

| Method | Description |
|---|---|
| `keymap:Register(mode, keyCode, commandName)` | Bind a key to a command in a mode |

### `keys` constants

`RIGHT`, `LEFT`, `DOWN`, `UP`, `ENTER`, `ESCAPE`, `BACKSPACE`, `HOME`, `END`

### `render`

| Method | Description |
|---|---|
| `render:SetCallback(fn)` | Set a Lua function called every frame after the editor renders but before the frame ends. Use for custom overlays. Pass `nil` to clear. |
| `render:Color(r, g, b, a?)` | Pack a color into an integer (default `a=255`). |
| `render:DrawRectangle(x, y, w, h, color)` | Draw a filled rectangle. `color` is a packed color integer or a `"#RRGGBB"` hex string. |
| `render:DrawText(text, x, y, size, color)` | Draw text at `(x, y)` with font `size`. |
| `render:DrawLine(x1, y1, x2, y2, color)` | Draw a line between two points. |
| `render:MeasureText(text, size)` | Returns text width in pixels. |
| `render:ScreenWidth()` | Current window width. |
| `render:ScreenHeight()` | Current window height. |

### `theme`

| Method | Description |
|---|---|
| `theme:SetColor(name, color)` | Set a theme color by name. `color` accepts packed integers or hex strings. |
| `theme:GetColor(name)` | Returns the packed color integer for a slot. |
| `theme:Names()` | Array of all configurable slot names. |

Slot names: `bg`, `text`, `gutter`, `cursor`, `selectBg`, `cursorLn`, `statusBg`, `statusTxt`.

### `statusline`

| Method | Description |
|---|---|
| `statusline:Set(fn)` | Replace the bottom status bar. `fn` receives no args and must return two strings: `left, right`. Pass `nil` to restore the default. |

Default status bar (left): `filename [+]`
Default status bar (right): `line:col/total -- MODE`

## Plugin directory

Any `*.lua` file placed in `~/.config/sumi/plugins/` is automatically executed at startup, after `init.lua`. Files are loaded in alphabetical order. One broken plugin does not stop others from loading.

Example: create `~/.config/sumi/plugins/timestamp.lua`:

```lua
commands:Register("insert_timestamp", "Insert current timestamp", 0, 0, function(e, args)
    local ts = os.date("%Y-%m-%d %H:%M:%S")
    for i = 1, #ts do
        e:InsertChar(string.byte(ts, i))
    end
end)
keymap:Register("normal", keys.F5, "insert_timestamp")
```

### `editor` settings

| Method | Description |
|---|---|
| `editor:SetSetting(name, value)` | Store a buffer-local setting. `value` can be boolean, number, string, or `nil`. |
| `editor:GetSetting(name)` | Retrieve a setting value, or `nil` if unset. |

Supported settings:

| Name | Type | Default | Effect |
|---|---|---|---|
| `line_numbers` | bool | `true` | Show line numbers in the gutter. When `false`, gutter shrinks to a thin separator. |
| `cursor_line` | bool | `true` | Highlight the line the cursor is on. |

### `events`

| Method | Description |
|---|---|
| `events:Register(name, fn)` | Register a Lua function to be called when an event fires. |
| `events:Unregister(name, fn?)` | Remove a specific handler, or all handlers for an event if `fn` is omitted. |

Event names:

| Name | When it fires | Arguments passed to handler |
|---|---|---|
| `file_open` | After a file is loaded | `(path)` |
| `save` | After a file is saved | `(path)` |
| `mode_change` | After the editor mode changes | `(mode_name)` e.g. `"normal"`, `"command"`, `"visual"` |
| `buffer_change` | After any buffer mutation (insert, delete, paste, undo, etc.) | none |

### `highlight`

| Method | Description |
|---|---|
| `highlight:SetCallback(fn)` | Register a syntax highlighter. `fn(line_idx, text)` receives 1-based line number and text. It must return an array of span tables: `{start, end, color}` or `{start=start, end=end, color=color}`. Positions are 1-based character indices. Color can be a packed integer from `render:Color(...)` or a `"#RRGGBB"` hex string. |

### `editor` search

| Method | Description |
|---|---|
| `editor:SetSearchPattern(pattern)` | Set the active search string. All visible matches are highlighted automatically. |
| `editor:SearchPattern()` | Get the current search string, or `""` if none. |
| `editor:ClearSearch()` | Remove the active search and clear highlights. |
| `editor:FindNext()` | Jump cursor to the next match. Returns `true` if found. |
| `editor:FindPrev()` | Jump cursor to the previous match. Returns `true` if found. |

Theme slot: `searchBg` (default: transparent yellow).

### `editor` error display

| Method | Description |
|---|---|
| `editor:ShowError(msg)` | Display a transient error message in the status bar. Auto-clears after 3 seconds. |
| `editor:ClearError()` | Dismiss the error immediately. |

Command errors ("pattern not found", "unsaved changes", etc.) are automatically shown in the status bar on the right side in red. No need to print to stderr.

Theme slot: `errorTxt` (default: red).

### `editor` tabs

| Method | Description |
|---|---|
| `editor:NewTab()` | Create a blank tab and switch to it. Returns 1-based tab index. |
| `editor:SwitchTab(idx)` | Switch to a tab by 1-based index. |
| `editor:CloseTab(idx)` | Close a tab by 1-based index. Returns the new active index. |
| `editor:NextTab()` | Switch to next tab (wraps around). |
| `editor:PrevTab()` | Switch to previous tab (wraps around). |
| `editor:TabCount()` | Number of open tabs. |
| `editor:TabNames()` | Array of tab labels (filepath + `[+]` if modified). |
| `editor:ActiveTab()` | 1-based index of the currently visible tab. |
| `editor:OpenFileInNewTab(path)` | Load a file into a new tab. |

Per-tab state (cursor, scroll, undo, settings, search pattern) is preserved when switching.

Default chords (handled in Go, not the keymap registry): `Cmd+T` new tab, `Cmd+W` close tab, `Cmd+PageDown` next tab, `Cmd+PageUp` previous tab.

## Architecture

| Layer | Language | Responsibility |
|---|---|---|
| Engine | Go | Buffer, cursor, undo, file I/O, render loop, input dispatch, OS clipboard, Lua bridge |
| Commands | Lua | Movement, editing, mode switching, file operations, undo — everything the editor does |
| Keymaps | Lua | All key bindings |
| Theme | Lua | All editor colors configurable via `theme:SetColor(...)`; defaults in Go |

The engine exposes primitives. The personality lives in Lua.

## `require()` — shared Lua modules

Standard Lua `require("modulename")` works. Sumi appends these search paths to `package.path`:

```
~/.config/sumi/lib/?.lua
~/.config/sumi/lib/?/init.lua
~/.config/sumi/plugins/?.lua
~/.config/sumi/plugins/?/init.lua
```

Create `~/.config/sumi/lib/utils.lua`:

```lua
local M = {}

function M.say_hello(name)
    print("Hello, " .. name)
end

return M
```

Use it anywhere (init.lua, plugins, or callbacks):

```lua
local utils = require("utils")
utils.say_hello("Sumi")
```

## Default Controls

| Key | Action |
|---|---|
| Arrows | Move cursor |
| Home / End | Start / end of line |
| Backspace | Delete before cursor |
| Enter | Insert newline |
| `:` | Enter command mode |
| `Esc` | Return to normal mode |
| `Cmd+V` | Enter visual mode |
| `d` (visual) | Delete selection |
| `y` (visual) | Yank to clipboard |
| `p` (normal) | Paste |
| `F3` | Find next match |
| `F4` | Find previous match |
| `F5` | Reload `init.lua` and plugins without quitting |
| `Cmd+T` | New blank tab |
| `Cmd+W` | Close current tab |
| `Cmd+PageDown` | Next tab |
| `Cmd+PageUp` | Previous tab |
| `Cmd+R` | Undo |
| `Cmd+S` | Save |
| Mouse click | Position cursor |
| Mouse drag | Visual selection |
| Double-click | Select word |
| Triple-click | Select line |
| Right-click | Extend selection |
| Scroll wheel | Scroll viewport |

## License

MIT
