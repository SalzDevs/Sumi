-- Default Sumi configuration
-- This is embedded in the binary and loaded on startup.
-- User config at ~/.config/sumi/init.lua can override these bindings.

-- =========================================================================
-- COMMANDS
-- =========================================================================
-- All editor behavior is defined here in Lua. The Go engine exposes
-- primitives; the personality lives in this file.

-- Movement
commands:Register("move_left", "Move cursor left", 0, 0, function(e, args)
    e.Cursor:MoveLeft()
end)

commands:Register("move_right", "Move cursor right", 0, 0, function(e, args)
    e.Cursor:MoveRight()
end)

commands:Register("move_up", "Move cursor up", 0, 0, function(e, args)
    e.Cursor:MoveUp()
end)

commands:Register("move_down", "Move cursor down", 0, 0, function(e, args)
    e.Cursor:MoveDown()
end)

-- Editing
commands:Register("backspace", "Delete character before cursor", 0, 0, function(e, args)
    e:Backspace()
end)

commands:Register("insert_newline", "Insert newline", 0, 0, function(e, args)
    e:InsertNewline()
end)

-- Mode switching
commands:Register("enter_command_mode", "Enter command mode", 0, 0, function(e, args)
    e:SetMode("command")
end)

commands:Register("cancel_command", "Cancel command mode", 0, 0, function(e, args)
    e:SetCommandLine("")
    e:SetMode("normal")
end)

commands:Register("command_backspace", "Delete last command character", 0, 0, function(e, args)
    e:CommandLineBackspace()
end)

commands:Register("enter_visual_mode", "Enter visual mode", 0, 0, function(e, args)
    e:EnterVisual()
end)

commands:Register("cancel_visual", "Cancel visual mode", 0, 0, function(e, args)
    e:ClearVisual()
end)

-- File commands
commands:Register("w", "Save the current file", 0, 0, function(e, args)
    local err = e:SaveFile()
    if err then return err end
end)

commands:Register("q", "Quit the editor", 0, 0, function(e, args)
    if e:Modified() then
        return "unsaved changes; use :q! to force"
    end
    e:Quit()
end)

commands:Register("q!", "Quit without saving", 0, 0, function(e, args)
    e:Quit()
end)

commands:Register("wq", "Save and quit", 0, 0, function(e, args)
    local err = e:SaveFile()
    if err then return err end
    if not e:Modified() then
        e:Quit()
    end
end)

commands:Register("e", "Open a file", 1, 1, function(e, args)
    local err = e:LoadFile(args[1])
    if err then return err end
end)

-- Undo
commands:Register("undo", "Undo last change", 0, 0, function(e, args)
    e:Undo()
end)

-- Line navigation
commands:Register("goto_line_start", "Go to start of line", 0, 0, function(e, args)
    e.Cursor:Goto(e.Cursor:Line(), 1)
end)

commands:Register("goto_line_end", "Go to end of line", 0, 0, function(e, args)
    e.Cursor:Goto(e.Cursor:Line(), 999999)
end)

-- =========================================================================
-- KEYMAPS
-- =========================================================================

-- Normal mode
keymap:Register("normal", keys.RIGHT, "move_right")
keymap:Register("normal", keys.LEFT, "move_left")
keymap:Register("normal", keys.DOWN, "move_down")
keymap:Register("normal", keys.UP, "move_up")
keymap:Register("normal", keys.BACKSPACE, "backspace")
keymap:Register("normal", keys.ENTER, "insert_newline")
keymap:Register("normal", keys.HOME, "goto_line_start")
keymap:Register("normal", keys.END, "goto_line_end")

-- Command mode
keymap:Register("command", keys.ESCAPE, "cancel_command")
keymap:Register("command", keys.BACKSPACE, "command_backspace")
keymap:Register("command", keys.ENTER, "execute_command")

-- Visual mode
keymap:Register("visual", keys.ESCAPE, "cancel_visual")
keymap:Register("visual", keys.LEFT, "move_left")
keymap:Register("visual", keys.RIGHT, "move_right")
keymap:Register("visual", keys.DOWN, "move_down")
keymap:Register("visual", keys.UP, "move_up")
keymap:Register("visual", keys.BACKSPACE, "backspace")

-- =========================================================================
-- EXAMPLES: Uncomment blocks below to explore the Lua API.
-- Copy anything you like into ~/.config/sumi/init.lua.
-- =========================================================================

-- -------------------------------------------------------------------------
-- Example 1: Theme customization
-- -------------------------------------------------------------------------
-- theme:SetColor("bg", "#1e1e1e")
-- theme:SetColor("text", "#d4d4d4")
-- theme:SetColor("cursor", "#ff0044")
-- theme:SetColor("selectBg", "#264f78")
-- theme:SetColor("cursorLn", "#2d2d2d")
-- theme:SetColor("statusBg", "#252526")
-- theme:SetColor("statusTxt", "#cccccc")
-- theme:SetColor("gutter", "#858585")

-- -------------------------------------------------------------------------
-- Example 2: Custom command — uppercase the current line
-- -------------------------------------------------------------------------
-- commands:Register("uppercase_line", "Uppercase current line", 0, 0, function(e, args)
--     local n = e.Cursor:Line()
--     e.Buffer:SetLine(n, string.upper(e.Buffer:GetLine(n)))
-- end)
-- keymap:Register("normal", keys.HOME, "uppercase_line")

-- -------------------------------------------------------------------------
-- Example 3: Custom command — go to a specific line
-- -------------------------------------------------------------------------
-- commands:Register("goto_line", "Go to line number", 1, 1, function(e, args)
--     local n = tonumber(args[1])
--     if n then
--         e.Cursor:Goto(n, 1)
--     end
-- end)
-- keymap:Register("normal", keys.END, "goto_line")
-- -- Usage: type ":goto_line 42" in command mode (or bind a key)

-- -------------------------------------------------------------------------
-- Example 4: Custom command — insert a timestamp
-- -------------------------------------------------------------------------
-- commands:Register("insert_timestamp", "Insert current timestamp", 0, 0, function(e, args)
--     e:InsertChar(string.byte("#"))
--     e:InsertNewline()
-- end)

-- -------------------------------------------------------------------------
-- Example 5: Render hook — mode indicator overlay
-- -------------------------------------------------------------------------
-- local RED   = render:Color(255, 80, 80)
-- local DARK  = "#1a1a1a"
-- render:SetCallback(function()
--     local w = render:ScreenWidth()
--     local text = "MODE: " .. string.upper(editor:Mode())
--     local tw = render:MeasureText(text, 14)
--     render:DrawRectangle(w - tw - 16, 6, tw + 12, 22, DARK)
--     render:DrawText(text, w - tw - 10, 9, 14, RED)
-- end)

-- -------------------------------------------------------------------------
-- Example 6: Render hook — toggleable overlay with a key
-- -------------------------------------------------------------------------
--[[
local overlayActive = false
local CYAN  = render:Color(0, 255, 255)
local BLACK = "#000000"

commands:Register("toggle_overlay", "Toggle info overlay", 0, 0, function(e, args)
    overlayActive = not overlayActive
    if overlayActive then
        render:SetCallback(function()
            local w = render:ScreenWidth()
            local h = render:ScreenHeight()
            local info = string.format("Line %d / %d", e.Cursor:Line(), e:LineCount())
            local iw = render:MeasureText(info, 12)
            render:DrawRectangle(4, h - 44, iw + 12, 18, BLACK)
            render:DrawText(info, 8, h - 42, 12, CYAN)
        end)
    else
        render:SetCallback(nil)
    end
end)

keymap:Register("normal", keys.F2, "toggle_overlay")
--]]

-- -------------------------------------------------------------------------
-- Example 7: Render hook — crosshair at screen center
-- -------------------------------------------------------------------------
--[[
local GREEN = render:Color(0, 255, 0)
render:SetCallback(function()
    local cx = render:ScreenWidth() / 2
    local cy = render:ScreenHeight() / 2
    render:DrawLine(cx - 6, cy, cx + 6, cy, GREEN)
    render:DrawLine(cx, cy - 6, cx, cy + 6, GREEN)
end)
--]]

-- -------------------------------------------------------------------------
-- Example 8: Custom statusline format
-- -------------------------------------------------------------------------
--[[
statusline:Set(function()
    local name = editor.Buffer:FilePath()
    if name == "" then name = "[No Name]" end
    if editor:Modified() then name = name .. " ●" end
    local left = name

    local right = string.format("Ln %d, Col %d  |  %s",
        editor.Cursor:Line(), editor.Cursor:Col(), string.upper(editor:Mode()))

    return left, right
end)
--]]

-- -------------------------------------------------------------------------
-- Example 9: Plugins — auto-loaded from ~/.config/sumi/plugins/*.lua
-- -------------------------------------------------------------------------
-- Drop .lua files into the plugins directory and they run on startup.
-- Example plugin: ~/.config/sumi/plugins/timestamp.lua
--
--     commands:Register("insert_timestamp", "Insert timestamp", 0, 0, function(e, args)
--         local ts = os.date("%Y-%m-%d %H:%M:%S")
--         for i = 1, #ts do
--             e:InsertChar(string.byte(ts, i))
--         end
--     end)
--     keymap:Register("normal", keys.F5, "insert_timestamp")

-- -------------------------------------------------------------------------
-- Example 10: Buffer-local settings
-- -------------------------------------------------------------------------
-- editor:SetSetting("line_numbers", false)   -- hide gutter numbers
-- editor:SetSetting("cursor_line", false)    -- disable current-line highlight

-- -------------------------------------------------------------------------
-- Example 11: Event hooks — react to editor state changes
-- -------------------------------------------------------------------------
--[[
events:Register("file_open", function(path)
    print("Opened: " .. path)
end)

events:Register("mode_change", function(mode)
    print("Mode changed to: " .. mode)
end)

events:Register("buffer_change", function()
    -- Track unsaved changes, update lint, etc.
end)
--]]

-- -------------------------------------------------------------------------
-- Example 12: Syntax highlighting — color numbers and strings
-- -------------------------------------------------------------------------
--[[
local ORANGE = render:Color(255, 165, 0)
local GREEN  = render:Color(80, 200, 120)

highlight:SetCallback(function(line_idx, text)
    local spans = {}
    -- Highlight numbers (sequence of digits)
    for start_pos, digits in text:gmatch("()(%d+)") do
        local end_pos = start_pos + #digits - 1
        table.insert(spans, {start_pos, end_pos, ORANGE})
    end
    -- Highlight quoted strings (simplistic — no escapes)
    for start_pos, content in text:gmatch('()("[^"]*")') do
        local end_pos = start_pos + #content - 1
        table.insert(spans, {start_pos, end_pos, GREEN})
    end
    return spans
end)
--]]

-- -------------------------------------------------------------------------
-- Example 13: Shared Lua modules via require()
-- -------------------------------------------------------------------------
-- Create ~/.config/sumi/lib/colors.lua:
--
--     local M = {}
--     M.keyword = render:Color(180, 120, 220)
--     M.comment = render:Color(120, 160, 120)
--     return M
--
-- Then use it in init.lua or any plugin:
--
--     local col = require("colors")
--     theme:SetColor("text", "#cccccc")

-- -------------------------------------------------------------------------
-- Search commands
-- -------------------------------------------------------------------------
commands:Register("search", "Set search pattern", 1, 1, function(e, args)
    e:SetSearchPattern(args[1])
end)

commands:Register("clear_search", "Clear search pattern", 0, 0, function(e, args)
    e:ClearSearch()
end)

commands:Register("find_next", "Jump to next search match", 0, 0, function(e, args)
    if not e:FindNext() then
        return "pattern not found"
    end
end)

commands:Register("find_prev", "Jump to previous search match", 0, 0, function(e, args)
    if not e:FindPrev() then
        return "pattern not found"
    end
end)

keymap:Register("normal", keys.F3, "find_next")
keymap:Register("normal", keys.F4, "find_prev")

-- -------------------------------------------------------------------------
-- Example 14: Search — type "/pattern" in command mode
-- -------------------------------------------------------------------------
--[[
commands:Register("slash_search", "Search for pattern", 1, 1, function(e, args)
    e:SetSearchPattern(args[1])
    e:FindNext()
end)

-- Bind '/' in normal mode to enter command mode with "/" pre-filled
commands:Register("enter_search_mode", "Enter search command mode", 0, 0, function(e, args)
    e:SetMode("command")
    e:SetCommandLine("/")
end)
keymap:Register("normal", 47, "enter_search_mode")  -- 47 is ASCII '/'
-- Then in command mode, Enter executes slash_search with the text after /
--]]

-- -------------------------------------------------------------------------
-- Reload config without restarting
-- -------------------------------------------------------------------------
commands:Register("reload_config", "Reload init.lua and plugins", 0, 0, function(e, args)
    -- This is a no-op placeholder; the Go side intercepts this command.
end)
keymap:Register("normal", keys.F5, "reload_config")

-- -------------------------------------------------------------------------
-- Tab commands
-- -------------------------------------------------------------------------
commands:Register("new_tab", "Open a new blank tab", 0, 0, function(e, args)
    e:NewTab()
end)

commands:Register("close_tab", "Close current tab", 0, 0, function(e, args)
    e:CloseTab(e:ActiveTab())
end)

commands:Register("next_tab", "Switch to next tab", 0, 0, function(e, args)
    e:NextTab()
end)

commands:Register("prev_tab", "Switch to previous tab", 0, 0, function(e, args)
    e:PrevTab()
end)

commands:Register("tabopen", "Open file in new tab", 1, 1, function(e, args)
    local err = e:OpenFileInNewTab(args[1])
    if err then return err end
end)

-- Tab chords are handled directly in Go (Cmd+T/W/PageUp/PageDown)
-- and do not go through the keymap registry.

-- -------------------------------------------------------------------------
-- Example 15: Display a custom error message in the UI
-- -------------------------------------------------------------------------
-- commands:Register("warn", "Show a warning", 1, 1, function(e, args)
--     e:ShowError("Warning: " .. args[1])
-- end)

-- -------------------------------------------------------------------------
-- Example 16: Tab bar overlay (requires render:SetCallback)
-- -------------------------------------------------------------------------
--[[
local WHITE = render:Color(255, 255, 255)
local GRAY  = "#444444"
local RED   = render:Color(255, 100, 100)
render:SetCallback(function()
    local tabs = editor:TabNames()
    local active = editor.ActiveTab + 1 -- would need API... skip for now
    -- (simplified: just list tabs at top)
end)
--]]

-- -------------------------------------------------------------------------
-- Example 17: Invert arrow keys (vim-like or just playful)
-- -------------------------------------------------------------------------
-- keymap:Register("normal", keys.LEFT, "move_right")
-- keymap:Register("normal", keys.RIGHT, "move_left")
-- keymap:Register("normal", keys.UP, "move_down")
-- keymap:Register("normal", keys.DOWN, "move_up")
