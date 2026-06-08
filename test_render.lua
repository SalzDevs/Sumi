-- Test config for the render API
-- Copy this to ~/.config/sumi/init.lua to try it out

local red   = render:Color(255, 0, 0)
local green = render:Color(0, 255, 0)
local dark  = "#1a1a1a"

render:SetCallback(function()
    local w = render:ScreenWidth()
    local text = "MODE: " .. string.upper(editor:Mode())
    local tw = render:MeasureText(text, 16)

    -- Red box + text in top-right corner
    render:DrawRectangle(w - tw - 24, 8, tw + 16, 28, dark)
    render:DrawText(text, w - tw - 16, 12, 16, red)

    -- Green crosshair center marker (tiny, just to prove lines work)
    local cx = w / 2
    local cy = render:ScreenHeight() / 2
    render:DrawLine(cx - 5, cy, cx + 5, cy, green)
    render:DrawLine(cx, cy - 5, cx, cy + 5, green)
end)
