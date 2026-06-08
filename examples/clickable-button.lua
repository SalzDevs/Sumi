-- Clickable button plugin for Sumi
-- ============================================================
-- Installation:
--   cp clickable-button.lua ~/.config/sumi/plugins/
--   ./sumi
--
-- Demonstrates mouse interaction inside a render hook.
-- Draws a "Click Me" button in the top-right corner.
-- Left-clicking it toggles a counter displayed on screen.

local BTN_X = 0
local BTN_Y = 30
local BTN_W = 80
local BTN_H = 24
local BTN_COLOR = "#3a3a3a"
local BTN_HOVER = "#555555"
local BTN_TEXT  = render:Color(200, 200, 200)

local counter = 0
local wasDown = false

render:SetCallback(function()
    local mx = render:MouseX()
    local my = render:MouseY()

    -- Compute button position (top-right, below tab bar)
    BTN_X = render:ScreenWidth() - BTN_W - 10

    -- Hit test
    local hovering = (mx >= BTN_X and mx <= BTN_X + BTN_W and
                      my >= BTN_Y and my <= BTN_Y + BTN_H)

    -- Detect click (pressed this frame, not held)
    local down = render:IsMouseDown(1)
    local clicked = (hovering and down and not wasDown)
    wasDown = down

    if clicked then
        counter = counter + 1
    end

    -- Draw button
    local color = hovering and BTN_HOVER or BTN_COLOR
    render:DrawRectangle(BTN_X, BTN_Y, BTN_W, BTN_H, color)
    render:DrawText("Click Me", BTN_X + 10, BTN_Y + 4, 14, BTN_TEXT)

    -- Draw counter
    local text = "Clicks: " .. counter
    local tw = render:MeasureText(text, 14)
    render:DrawText(text, BTN_X - tw - 10, BTN_Y + 4, 14, BTN_TEXT)
end)

print("Clickable button plugin loaded")
