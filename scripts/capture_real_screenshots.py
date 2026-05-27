import subprocess
import re
from PIL import Image, ImageDraw, ImageFont
import time

TERMINAL_BG = (30, 30, 46)

ANSI_REGEX = re.compile(r'\x1b\[([0-9;]*)m')

def parse_ansi_color(parts):
    """Parse ANSI color codes into (r,g,b) or None."""
    if not parts:
        return None
    
    i = 0
    while i < len(parts):
        code = parts[i]
        if code == '38' and i + 1 < len(parts):
            if parts[i + 1] == '2' and i + 4 < len(parts):
                return (int(parts[i + 2]), int(parts[i + 3]), int(parts[i + 4]))
            elif parts[i + 1] == '5' and i + 2 < len(parts):
                return ansi256_to_rgb(int(parts[i + 2]))
            i += 1
        elif code == '48':
            i += 1
        i += 1
    return None

def ansi256_to_rgb(n):
    if n < 8:
        vals = [0, 205, 255]
        r = vals[(n >> 2) & 1] * ((n >> 2) & 1) or vals[0]
        g = vals[(n >> 1) & 1] * ((n >> 1) & 1) or vals[0]
        b = vals[n & 1] * (n & 1) or vals[0]
        return (r, g, b)
    elif n < 16:
        m = n - 8
        r = 127 + 128 * ((m >> 2) & 1)
        g = 127 + 128 * ((m >> 1) & 1)
        b = 127 + 128 * (m & 1)
        return (r, g, b)
    elif n < 232:
        n -= 16
        r = int((n // 36) * 85 / 3)
        g = int(((n // 6) % 6) * 85 / 3)
        b = int((n % 6) * 85 / 3)
        return (r, g, b)
    else:
        v = int((n - 232) * 255 / 23 + 8)
        return (v, v, v)

def parse_ansi_line(line):
    """Parse a line with ANSI codes into segments with colors."""
    segments = []
    pos = 0
    current_fg = None
    current_bg = None
    current_bold = False
    
    for match in ANSI_REGEX.finditer(line):
        if match.start() > pos:
            text = line[pos:match.start()]
            segments.append((text, current_fg, current_bg, current_bold))
        
        codes = match.group(1)
        if not codes:
            current_fg = None
            current_bg = None
            current_bold = False
        else:
            parts = codes.split(';')
            for p in parts:
                if p == '0':
                    current_fg = None
                    current_bg = None
                    current_bold = False
                elif p == '1':
                    current_bold = True
                elif p == '22':
                    current_bold = False
                elif p == '39':
                    current_fg = None
                elif p == '49':
                    current_bg = None
                elif p.startswith('38'):
                    pass
                elif p.startswith('48'):
                    pass
            
            color = parse_ansi_color(parts)
            if color:
                if '38' in parts and parts[parts.index('38') + 1] == '2':
                    current_fg = color
                if '48' in parts and parts[parts.index('48') + 1] == '2':
                    current_bg = color
        
        pos = match.end()
    
    if pos < len(line):
        segments.append((line[pos:], current_fg, current_bg, current_bold))
    
    return segments

def render_ansi_to_image(lines, width=900, height=560, font_size=13):
    font = ImageFont.truetype("/System/Library/Fonts/Monaco.ttf", font_size)
    bbox = font.getbbox("M")
    char_w = bbox[2] - bbox[0]
    char_h = bbox[3] - bbox[1]
    line_height = int(char_h * 1.35)
    
    padding_x = 20
    padding_y = 36
    
    img = Image.new("RGB", (width, height), TERMINAL_BG)
    draw = ImageDraw.Draw(img)
    
    draw.rectangle([0, 0, width, 26], fill=(49, 50, 68))
    draw.ellipse([10, 8, 20, 18], fill=(255, 95, 86))
    draw.ellipse([26, 8, 36, 18], fill=(255, 189, 46))
    draw.ellipse([42, 8, 52, 18], fill=(39, 201, 63))
    
    y = padding_y
    for line in lines:
        segments = parse_ansi_line(line)
        x = padding_x
        for text, fg, bg, bold in segments:
            if not text:
                continue
            
            bg_color = bg if bg else TERMINAL_BG
            fg_color = fg if fg else (249, 250, 251)
            
            if bold and fg:
                fg_color = tuple(min(255, c + 30) for c in fg)
            
            for ch in text:
                ch_bbox = font.getbbox(ch)
                ch_w = ch_bbox[2] - ch_bbox[0] if ch_bbox else char_w
                
                if bg and bg != TERMINAL_BG:
                    draw.rectangle([x, y - 2, x + ch_w + 1, y + line_height - 1], fill=bg_color)
                
                draw.text((x, y), ch, fill=fg_color, font=font)
                x += ch_w
        
        y += line_height
    
    return img

def capture_tmux_state(session_name):
    result = subprocess.run(
        ["tmux", "capture-pane", "-e", "-t", session_name, "-p"],
        capture_output=True, text=True
    )
    return result.stdout.splitlines()

def run_and_capture(state_name, keys="", delay=2):
    subprocess.run(["tmux", "new-session", "-d", "-s", "overseer_cap",
                   "cd /Users/david.lopes/.config/overseer-personal/worktrees/78ee833c && ./bin/overseer"],
                  capture_output=True)
    
    time.sleep(2)
    
    if keys:
        for key in keys:
            if key == "Escape":
                subprocess.run(["tmux", "send-keys", "-t", "overseer_cap", "Escape"],
                              capture_output=True)
            else:
                subprocess.run(["tmux", "send-keys", "-t", "overseer_cap", key],
                              capture_output=True)
            time.sleep(0.5)
        time.sleep(0.5)
    
    lines = capture_tmux_state("overseer_cap")
    
    subprocess.run(["tmux", "send-keys", "-t", "overseer_cap", "q"],
                  capture_output=True)
    time.sleep(0.5)
    subprocess.run(["tmux", "kill-session", "-t", "overseer_cap"],
                  capture_output=True)
    
    return lines

def main():
    print("Capturing dashboard...")
    lines = run_and_capture("dashboard")
    img = render_ansi_to_image(lines)
    img.save("/Users/david.lopes/.config/overseer-personal/worktrees/78ee833c/docs/screenshots/01-dashboard.png")
    print("Saved 01-dashboard.png")
    
    print("Capturing create form...")
    lines = run_and_capture("create", keys=["n"])
    img = render_ansi_to_image(lines)
    img.save("/Users/david.lopes/.config/overseer-personal/worktrees/78ee833c/docs/screenshots/02-create-session.png")
    print("Saved 02-create-session.png")
    
    print("Capturing help menu...")
    lines = run_and_capture("help", keys=["?"])
    img = render_ansi_to_image(lines)
    img.save("/Users/david.lopes/.config/overseer-personal/worktrees/78ee833c/docs/screenshots/04-help-menu.png")
    print("Saved 04-help-menu.png")
    
    print("All screenshots captured from real app!")

if __name__ == "__main__":
    main()
