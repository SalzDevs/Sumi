-- Default Sumi configuration
-- This is embedded in the binary and loaded on startup.
-- User config at ~/.config/sumi/init.lua can override these bindings.

-- -------------------------------------------------------------------------
-- Commands (registered in Lua — the SDK in action)
-- -------------------------------------------------------------------------

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

-- -------------------------------------------------------------------------
-- Keymaps
-- -------------------------------------------------------------------------

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

-- -------------------------------------------------------------------------
-- Theme customization (uncomment to override default colors)
-- -------------------------------------------------------------------------
theme:SetColor("bg", "#1e1e1e")
theme:SetColor("text", "#d4d4d4")
theme:SetColor("cursor", "#ff0044")
theme:SetColor("selectBg", "#264f78")
theme:SetColor("cursorLn", "#2d2d2d")
theme:SetColor("statusBg", "#252526")
theme:SetColor("statusTxt", "#cccccc")
theme:SetColor("gutter", "#858585")

-- -------------------------------------------------------------------------
-- Render hook example (uncomment to enable a custom overlay)
-- -------------------------------------------------------------------------
--[[
local red = render:Color(255, 0, 0)
local dark = "#1a1a1a"
render:SetCallback(function()
    local w = render:ScreenWidth()
    local text = "Mode: " .. editor:Mode()
    local tw = render:MeasureText(text, 16)
    render:DrawRectangle(w - tw - 20, 10, tw + 16, 24, dark)
    render:DrawText(text, w - tw - 12, 14, 16, red)
end)
--]]
