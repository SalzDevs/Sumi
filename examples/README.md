# Sumi Example Plugins

These plugins demonstrate what the Sumi Lua API can do. Each is a single
`.lua` file you can drop into `~/.config/sumi/plugins/` and restart (or
press `F5` to reload).

## rainbow-parens.lua

Colors matching `(` and `)` in rainbow order. Uses `highlight:SetCallback`
to return color spans for each visible line.

**APIs used:** `highlight`, `editor.Buffer:GetLine`, `render:Color`

## minimap.lua

Draws a zoomed-out buffer overview on the right side of the screen.
Shows line lengths as horizontal strips, cursor position, and viewport
highlight.

**APIs used:** `render:SetCallback`, `render:DrawRectangle`,
`render:ScreenWidth`, `render:ScreenHeight`, `editor:LineCount`,
`editor.Buffer:GetLine`, `editor.Cursor:Line`, `editor:ViewportScrollY`

## Writing your own

The full API surface is documented in the main README. Common patterns:

```lua
-- React to state changes
events:Register("buffer_change", function()
    -- update your plugin state
end)

-- Draw custom UI every frame
render:SetCallback(function()
    local w = render:ScreenWidth()
    local h = render:ScreenHeight()
    render:DrawRectangle(10, 10, 100, 20, "#333333")
    render:DrawText("Hello", 14, 14, 16, render:Color(255, 255, 255))
end)

-- Colorize text
highlight:SetCallback(function(line_idx, text)
    local spans = {}
    for i = 1, #text do
        local c = text:sub(i, i)
        if c:match("%d") then
            table.insert(spans, {i, i, render:Color(255, 165, 0)})
        end
    end
    return spans
end)

-- Add commands and keymaps
commands:Register("my_cmd", "Do something", 0, 0, function(e, args)
    e:ShowError("It works!")
end)
keymap:Register("normal", keys.F2, "my_cmd")
```

Plugins can share code via `require()` if you place modules in
`~/.config/sumi/lib/`.
