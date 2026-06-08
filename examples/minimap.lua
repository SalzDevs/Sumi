-- Minimap plugin for Sumi
-- ============================================================
-- Installation:
--   cp minimap.lua ~/.config/sumi/plugins/
--   ./sumi
--
-- Shows a zoomed-out overview of the current buffer on the right side
-- of the screen. The active viewport is highlighted. Each line is drawn
-- as a thin horizontal strip; longer lines are wider.

local MINIMAP_W = 100
local PAD = 6
local CHAR_W = 2
local LINE_H = 2
local BG = "#1a1a1a"
local TEXT = render:Color(80, 80, 80)
local CURSOR = render:Color(200, 200, 200)
local VIEW_BG = render:Color(60, 100, 160, 100)

local function maxLineLen(e)
    local m = 0
    for i = 1, e:LineCount() do
        local len = #e.Buffer:GetLine(i)
        if len > m then m = len end
    end
    return m
end

render:SetCallback(function()
    local e = editor
    local screenW = render:ScreenWidth()
    local screenH = render:ScreenHeight()

    local bufLines = e:LineCount()
    if bufLines == 0 then return end

    local longest = maxLineLen(e)
    local mapW = longest * CHAR_W
    local mapH = bufLines * LINE_H

    -- Scale to fit the fixed minimap width
    local scale = 1.0
    if mapW > MINIMAP_W - PAD * 2 then
        scale = (MINIMAP_W - PAD * 2) / mapW
    end

    local drawW = math.min(mapW * scale + PAD * 2, MINIMAP_W)
    local drawH = mapH * scale
    local x = screenW - drawW - PAD
    local y = (screenH - drawH) / 2
    if y < PAD then y = PAD end

    -- Background panel
    render:DrawRectangle(x, y, drawW, drawH, BG)

    -- Each line as a horizontal strip
    local innerX = x + PAD
    local innerY = y + PAD
    for lineIdx = 1, bufLines do
        local line = e.Buffer:GetLine(lineIdx)
        local len = #line
        local ly = innerY + (lineIdx - 1) * LINE_H * scale
        local lw = len * CHAR_W * scale
        local h = math.max(1, LINE_H * scale)
        if lw > 0 then
            render:DrawRectangle(innerX, ly, lw, h, TEXT)
        end
        -- Cursor line glow
        if lineIdx == e.Cursor:Line() then
            render:DrawRectangle(innerX, ly, math.max(lw, 4), h + 1, CURSOR)
        end
    end

    -- Viewport highlight
    local scrollY = e:ViewportScrollY()
    local visibleLines = math.floor((screenH - 24) / 20)
    local vy = innerY + scrollY * LINE_H * scale
    local vh = visibleLines * LINE_H * scale
    render:DrawRectangle(x, vy, drawW, vh, VIEW_BG)
end)

print("Minimap plugin loaded")
