-- Default Sumi configuration
-- This is embedded in the binary and loaded on startup.
-- User config at ~/.config/sumi/init.lua can override these bindings.

-- Normal mode
keymap:Register("normal", keys.RIGHT, "move_right")
keymap:Register("normal", keys.LEFT, "move_left")
keymap:Register("normal", keys.DOWN, "move_down")
keymap:Register("normal", keys.UP, "move_up")
keymap:Register("normal", keys.BACKSPACE, "backspace")
keymap:Register("normal", keys.ENTER, "insert_newline")

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
