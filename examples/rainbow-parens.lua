-- Rainbow Parentheses plugin for Sumi
-- ============================================================
-- Installation:
--   cp rainbow-parens.lua ~/.config/sumi/plugins/
--   ./sumi
--
-- Colors matching brackets in rainbow order. Proves that the
-- highlight:SetCallback API works for real plugin development.

local COLORS = {
    render:Color(255, 100, 100),  -- red
    render:Color(100, 255, 100),  -- green
    render:Color(100, 100, 255),  -- blue
    render:Color(255, 255, 100),  -- yellow
    render:Color(255, 100, 255),  -- magenta
    render:Color(100, 255, 255),  -- cyan
}

-- Count bracket depth up to (but not including) the given line.
-- This is cheap for small files; for large files a plugin would
-- cache the result incrementally via events:buffer_change.
local function bracketDepthBefore(targetLine)
    local depth = 0
    for lineIdx = 1, targetLine - 1 do
        local line = editor.Buffer:GetLine(lineIdx)
        for c in line:gmatch("[()]") do
            if c == "(" then
                depth = depth + 1
            else
                depth = math.max(0, depth - 1)
            end
        end
    end
    return depth
end

highlight:SetCallback(function(line_idx, text)
    local spans = {}
    local depth = bracketDepthBefore(line_idx)

    for i = 1, #text do
        local c = text:sub(i, i)
        if c == "(" then
            depth = depth + 1
            local color = COLORS[(depth - 1) % #COLORS + 1]
            table.insert(spans, {i, i, color})
        elseif c == ")" then
            local color = COLORS[(depth - 1) % #COLORS + 1]
            table.insert(spans, {i, i, color})
            depth = math.max(0, depth - 1)
        end
    end

    return spans
end)

print("Rainbow Parentheses plugin loaded")
