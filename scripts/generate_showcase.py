#!/usr/bin/env python3
"""
Ventox All-in-One Showcase Video Generator
Renders a 1080p (1920x1080 @ 30fps) realistic macOS/Linux terminal recording
showcasing ALL features of Ventox with frame-synchronized sound effects (SFX):
  - Mechanical keyboard typing clicks (Cherry MX style)
  - Deep enter key thocks
  - Crystal dings for checkmarks, bypasses & formatting
  - Data transfer humming audio for the live streaming progress bar
  - Triumphant victory chords
  - Ambient lo-fi background music
Outputs:
  - /home/panda/Desktop/ventox-showcase.mp4
  - /home/panda/Desktop/ventox-showcase.gif
  - /home/panda/Desktop/ventox/ventox-showcase.mp4
  - /home/panda/Desktop/ventox/ventox-showcase.gif
"""

import math
import os
import random
import shutil
import struct
import subprocess
import sys
import wave
from PIL import Image, ImageDraw, ImageFont

WIDTH, HEIGHT = 1920, 1080
FPS = 30
TOTAL_FRAMES = 1140  # 38 seconds at 30 fps
TOTAL_SECONDS = TOTAL_FRAMES / FPS
SAMPLE_RATE = 44100

DEST_DIR_REPO = "/home/panda/Desktop/ventox"
DEST_DIR_DESKTOP = "/home/panda/Desktop"

TEMP_RAW_VIDEO = "/tmp/ventox_raw_video.mp4"
TEMP_AUDIO = "/tmp/ventox_mastered_audio.wav"
FINAL_MP4 = os.path.join(DEST_DIR_REPO, "ventox-showcase.mp4")
FINAL_GIF = os.path.join(DEST_DIR_REPO, "ventox-showcase.gif")

# Fonts
FONT_MONO = "/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf"
FONT_MONO_BOLD = "/usr/share/fonts/truetype/liberation/LiberationMono-Bold.ttf"
FONT_SANS = "/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf"

def load_font(path, size):
    try:
        return ImageFont.truetype(path, size)
    except Exception:
        return ImageFont.load_default()

font_term = load_font(FONT_MONO, 20)
font_term_bold = load_font(FONT_MONO_BOLD, 20)
font_title = load_font(FONT_SANS, 15)

# Color Palette
C_BG_DESKTOP = (10, 14, 23)
C_TERM_BG = (15, 20, 31)
C_TITLEBAR_BG = (26, 34, 52)
C_BORDER = (45, 58, 85)

C_GREEN = (74, 222, 128)
C_CYAN = (56, 189, 248)
C_YELLOW = (250, 204, 21)
C_RED = (248, 113, 113)
C_GRAY = (148, 163, 184)
C_DARK_GRAY = (71, 85, 105)
C_WHITE = (241, 245, 249)
C_MAGENTA = (232, 121, 249)

TERM_X, TERM_Y = 160, 90
TERM_W, TERM_H = 1600, 900
TITLE_H = 46
CONTENT_H = TERM_H - TITLE_H

START_X = TERM_X + 32
START_Y = TERM_Y + TITLE_H + 24
LINE_HEIGHT = 28
MAX_LINES = 29

def draw_window_frame():
    base = Image.new("RGB", (WIDTH, HEIGHT), C_BG_DESKTOP)
    draw = ImageDraw.Draw(base)

    # Subtle glowing wallpaper aura
    for r in range(400, 0, -40):
        alpha = int((1 - r / 400) * 18)
        color = (15 + alpha, 23 + alpha * 2, 42 + alpha * 3)
        draw.ellipse([WIDTH // 2 - r * 2, HEIGHT // 2 - r, WIDTH // 2 + r * 2, HEIGHT // 2 + r], fill=color)

    # Terminal shadow
    for offset in range(16, 0, -2):
        s_box = [TERM_X - offset, TERM_Y - offset // 2, TERM_X + TERM_W + offset, TERM_Y + TERM_H + offset]
        draw.rounded_rectangle(s_box, radius=18, fill=(4, 7, 12))

    # Window body & border
    term_box = [TERM_X, TERM_Y, TERM_X + TERM_W, TERM_Y + TERM_H]
    draw.rounded_rectangle(term_box, radius=14, fill=C_TERM_BG, outline=C_BORDER, width=2)

    # Title bar
    title_box = [TERM_X, TERM_Y, TERM_X + TERM_W, TERM_Y + TITLE_H]
    draw.rounded_rectangle(title_box, radius=14, fill=C_TITLEBAR_BG)
    draw.rectangle([TERM_X, TERM_Y + TITLE_H - 14, TERM_X + TERM_W, TERM_Y + TITLE_H], fill=C_TITLEBAR_BG)
    draw.line([(TERM_X, TERM_Y + TITLE_H), (TERM_X + TERM_W, TERM_Y + TITLE_H)], fill=C_BORDER, width=1)

    # Traffic light buttons
    draw.ellipse([TERM_X + 22, TERM_Y + 16, TERM_X + 36, TERM_Y + 30], fill=(239, 68, 68))
    draw.ellipse([TERM_X + 46, TERM_Y + 16, TERM_X + 60, TERM_Y + 30], fill=(234, 179, 8))
    draw.ellipse([TERM_X + 70, TERM_Y + 16, TERM_X + 84, TERM_Y + 30], fill=(34, 197, 94))

    # Title label
    title_text = "bash - ventox v0.2.0 • Cross-Platform Fast Bootable USB CLI"
    bbox = draw.textbbox((0, 0), title_text, font=font_title)
    tx = TERM_X + (TERM_W - (bbox[2] - bbox[0])) // 2
    draw.text((tx, TERM_Y + 14), title_text, font=font_title, fill=C_GRAY)

    return base

def get_terminal_content(frame):
    lines = []

    def prompt(cmd_text, show_cursor=True):
        p = [
            ("developer@workstation", C_GREEN, True),
            (":", C_GRAY, False),
            ("~/ventox", C_CYAN, True),
            ("$ ", C_WHITE, True),
            (cmd_text, C_WHITE, False)
        ]
        if show_cursor and (frame // 10) % 2 == 0:
            p.append(("█", C_CYAN, False))
        return p

    def type_anim(full_cmd, start_f, end_f):
        if frame < start_f:
            return ""
        if frame >= end_f:
            return full_cmd
        chars = int((frame - start_f) / (end_f - start_f) * len(full_cmd))
        return full_cmd[:chars]

    # === SECTION 1: ventox list (Frames 0 - 150) ===
    if frame >= 45:
        c1 = type_anim("ventox list", 45, 85)
        lines.append(prompt(c1, show_cursor=(frame < 85)))
    else:
        lines.append(prompt("", show_cursor=True))

    if frame >= 85:
        lines.append([("VENTOX - Discovering block storage devices and local OS images...", C_GRAY, False)])
        lines.append([("", C_WHITE, False)])

    if frame >= 105:
        lines.append([("AVAILABLE STORAGE DEVICES:", C_CYAN, True)])
        lines.append([("  [1] /dev/sdb  ", C_YELLOW, True), ("SanDisk Ultra 3.0 (32.0 GB)  ", C_WHITE, False), ("[Removable USB]", C_GREEN, False)])
        lines.append([("      Protection : System disk protection ACTIVE (Safe to use)", C_GRAY, False)])
        lines.append([("  [*] /dev/nvme0n1 Samsung 990 PRO (2.0 TB) ", C_DARK_GRAY, False), ("[SYSTEM DISK - LOCKED]", C_RED, True)])
        lines.append([("", C_WHITE, False)])

    if frame >= 120:
        lines.append([("DISCOVERED OS INSTALLATION IMAGES:", C_CYAN, True)])
        lines.append([("  • Win11_24H2_English_x64.iso  ", C_WHITE, True), ("(6.24 GB)  ", C_GRAY, False), ("[Windows 11 UEFI]", C_MAGENTA, False)])
        lines.append([("  • ubuntu-24.04.1-desktop.iso   ", C_WHITE, False), ("(5.78 GB)  ", C_GRAY, False), ("[Linux Hybrid ISO]", C_YELLOW, False)])
        lines.append([("", C_WHITE, False)])

    # === SECTION 2: ventox --dry-run (Frames 150 - 290) ===
    if frame >= 150:
        c2 = type_anim("ventox --dry-run --iso win11.iso --device /dev/sdb", 165, 210)
        lines.append(prompt(c2, show_cursor=(frame < 210)))

    if frame >= 210:
        lines.append([("VENTOX - Running Preflight Validation (Dry-Run Simulation)", C_YELLOW, True)])
        lines.append([("  ✔ Verifying target device: /dev/sdb is removable USB storage", C_GREEN, False)])
        lines.append([("  ✔ Target capacity 32.0 GB >= required 6.24 GB", C_GREEN, False)])
        lines.append([("  ✔ System disk guard: /dev/sdb is NOT a system or boot drive", C_GREEN, False)])
        lines.append([("  ✔ Selected Architecture: Rufus Dual-Partition (FAT32 Boot + NTFS Data)", C_CYAN, False)])
        lines.append([("  ✔ Windows 11 Bypasses: TPM 2.0, SecureBoot, RAM & Local Account READY", C_MAGENTA, False)])
        lines.append([("DRY-RUN COMPLETE: Target is safe, planned operations validated. No data modified.", C_GREEN, True)])
        lines.append([("", C_WHITE, False)])

    # === SECTION 3: Real Burning (Frames 290 - 750) ===
    if frame >= 290:
        c3 = type_anim("sudo ventox --iso win11.iso --device /dev/sdb --verify", 315, 365)
        lines.append(prompt(c3, show_cursor=(frame < 365)))

    if frame >= 365:
        lines.append([("[1/5] Unmounting target partitions and zeroing MBR/GPT headers...", C_GRAY, False)])
    if frame >= 385:
        lines.append([("      ✔ Zeroed sector 0-2048 on /dev/sdb", C_GREEN, False)])
    if frame >= 405:
        lines.append([("[2/5] Creating Rufus-Style Dual-Partition Table on /dev/sdb...", C_CYAN, True)])
        lines.append([("      Partition 1: 100 MB FAT32 [UEFI_BOOT] (Active/Bootable)", C_WHITE, False)])
        lines.append([("      Partition 2: 31.9 GB NTFS   [WININSTALL] (Install & Large WIM)", C_WHITE, False)])
    if frame >= 425:
        lines.append([("[3/5] Injecting Windows 11 Bypass Configuration (autounattend.xml)...", C_MAGENTA, True)])
        lines.append([("      ✔ Bypass TPM 2.0 & SecureBoot checks", C_GREEN, False)])
        lines.append([("      ✔ Bypass 4GB RAM & Storage requirement checks", C_GREEN, False)])
        lines.append([("      ✔ Bypass Mandatory Microsoft Account (Enable Local Account)", C_GREEN, False)])
        lines.append([("      ✔ Injected EFI Stub: /Volumes/UEFI_BOOT/EFI/BOOT/BOOTX64.EFI", C_GREEN, False)])

    if frame >= 440:
        prog_f = min(frame, 660) - 440
        pct = min(100.0, (prog_f / 220.0) * 100.0)
        copied_gb = (pct / 100.0) * 6.24
        speed = 142.5 + 8.0 * math.sin(frame * 0.2)
        bar_len = 32
        filled = int((pct / 100.0) * bar_len)
        bar_str = "█" * filled + "░" * (bar_len - filled)
        lines.append([("[4/5] Streaming ISO filesystem and large install.wim...", C_CYAN, True)])
        lines.append([
            (f"      [{bar_str}] ", C_GREEN, True),
            (f"{pct:5.1f}% ", C_YELLOW, True),
            (f"({copied_gb:.2f}/6.24 GB) ", C_WHITE, False),
            (f"@ {speed:.1f} MB/s", C_CYAN, False)
        ])

    if frame >= 660:
        lines.append([("      ✔ Data written & filesystem cache flushed to disk.", C_GREEN, False)])
    if frame >= 685:
        lines.append([("[5/5] Streaming SHA-256 verification...", C_CYAN, True)])
        lines.append([("      Source ISO : e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", C_GRAY, False)])
        lines.append([("      Target USB : e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", C_GRAY, False)])
        lines.append([("      ✔ CHECKSUM MATCHED (100% Bit-accurate verification)", C_GREEN, True)])
    if frame >= 705:
        lines.append([("", C_WHITE, False)])
        lines.append([("╔══════════════════════════════════════════════════════════════════════╗", C_GREEN, True)])
        lines.append([("║  ✔ SUCCESS: Windows 11 UEFI Bootable USB is Ready to Boot!           ║", C_GREEN, True)])
        lines.append([("╚══════════════════════════════════════════════════════════════════════╝", C_GREEN, True)])
        lines.append([("", C_WHITE, False)])

    # === SECTION 4: Standalone Format (Frames 750 - 930) ===
    if frame >= 750:
        c4 = type_anim("sudo ventox format --device /dev/sdb --fs exfat --label \"MY_USB\"", 775, 835)
        lines.append(prompt(c4, show_cursor=(frame < 835)))

    if frame >= 835:
        lines.append([("VENTOX - Standalone Multi-Filesystem Drive Formatter", C_CYAN, True)])
        lines.append([("  • Target Device : /dev/sdb (SanDisk Ultra 3.0)", C_WHITE, False)])
        lines.append([("  • Filesystem    : exFAT (Native cross-platform)", C_YELLOW, False)])
        lines.append([("  • Volume Label  : MY_USB", C_WHITE, False)])
    if frame >= 865:
        lines.append([("  ✔ Erasing partition table & creating GPT layout...", C_GRAY, False)])
    if frame >= 885:
        lines.append([("  ✔ Formatting drive to exFAT with cluster size 128 KB...", C_CYAN, False)])
        lines.append([("  ✔ Formatted successfully in 1.42s! Volume mounted at /media/MY_USB", C_GREEN, True)])
        lines.append([("", C_WHITE, False)])

    # === SECTION 5: Version Matrix (Frames 930 - End) ===
    if frame >= 930:
        c5 = type_anim("ventox version", 955, 985)
        lines.append(prompt(c5, show_cursor=(frame < 985)))

    if frame >= 985:
        lines.append([("ventox v0.2.0 (built with go1.24 - linux/amd64)", C_CYAN, True)])
        lines.append([("Cross-Platform Support Matrix:", C_WHITE, True)])
        lines.append([("  • Linux   : x86_64, arm64 (Ubuntu, Fedora, Arch, Debian)", C_GREEN, False)])
        lines.append([("  • macOS   : Apple Silicon (M1/M2/M3/M4) & Intel x86_64", C_GREEN, False)])
        lines.append([("  • Windows : Windows 10/11 x64, ARM64 (Rufus-compatible)", C_GREEN, False)])
        lines.append([("Repository: https://github.com/0xzahed/ventox (MIT Licensed)", C_YELLOW, False)])
        lines.append([("", C_WHITE, False)])
        lines.append(prompt("", show_cursor=True))

    return lines

def synthesize_audio():
    print("Synthesizing authentic mechanical keyboard typing sound effects & background ambience...")
    total_samples = int(SAMPLE_RATE * TOTAL_SECONDS)
    left = [0.0] * total_samples
    right = [0.0] * total_samples

    def add_sample(idx, val_l, val_r):
        if 0 <= idx < total_samples:
            left[idx] += val_l
            right[idx] += val_r

    def add_mech_keystroke(time_sec, char='a', is_enter=False):
        is_space = (char == ' ')
        start_idx = int(time_sec * SAMPLE_RATE)

        if is_enter:
            res_freq = 330.0
            thump_freq = 145.0
            dur = 0.075
            vol = 0.95
        elif is_space:
            res_freq = 380.0
            thump_freq = 175.0
            dur = 0.065
            vol = 0.90
        else:
            res_freq = random.uniform(560.0, 780.0)
            thump_freq = random.uniform(220.0, 290.0)
            dur = 0.050
            vol = random.uniform(0.82, 0.94)

        pan = 0.5 + random.uniform(-0.15, 0.15)
        num_s = int(dur * SAMPLE_RATE)

        # 1. Downstroke: Crisp click leaf + PBT keycap hollow body + Case plate thump
        for i in range(num_s):
            t = i / SAMPLE_RATE
            click = (random.random() * 2.0 - 1.0) * math.exp(-380.0 * t) * 0.75
            body = (
                math.sin(2 * math.pi * res_freq * t) +
                0.55 * math.sin(2 * math.pi * res_freq * 1.85 * t) +
                0.25 * math.sin(2 * math.pi * res_freq * 2.8 * t)
            ) * math.exp(-110.0 * t) * 0.55
            thump = (
                math.sin(2 * math.pi * thump_freq * t) +
                0.4 * math.sin(2 * math.pi * thump_freq * 0.5 * t)
            ) * math.exp(-65.0 * t) * 0.50
            stab = 0.0
            if is_space or is_enter:
                stab = math.sin(2 * math.pi * 1850.0 * t) * math.exp(-250.0 * t) * 0.20

            s = (click + body + thump + stab) * vol
            add_sample(start_idx + i, s * (1.0 - pan), s * pan)

        # 2. Upstroke Release: Plastic slider bouncing against switch top housing
        up_offset = int((0.026 + random.uniform(0.008, 0.020)) * SAMPLE_RATE)
        up_idx = start_idx + up_offset
        up_dur = int(0.025 * SAMPLE_RATE)
        up_freq = res_freq * 1.5 + random.uniform(-40, 40)
        for i in range(up_dur):
            t = i / SAMPLE_RATE
            up_snap = (random.random() * 2.0 - 1.0) * math.exp(-420.0 * t) * 0.40
            up_tone = math.sin(2 * math.pi * up_freq * t) * math.exp(-220.0 * t) * 0.28
            s_up = (up_snap + up_tone) * vol * 0.45
            add_sample(up_idx + i, s_up * (1.0 - pan), s_up * pan)

    def add_ding(time_sec, freq=1046.5, volume=0.28, duration=0.6):
        start_idx = int(time_sec * SAMPLE_RATE)
        num_s = int(duration * SAMPLE_RATE)
        for i in range(num_s):
            t = i / SAMPLE_RATE
            env = math.exp(-6.5 * t)
            s = (
                math.sin(2.0 * math.pi * freq * t) +
                0.45 * math.sin(2.0 * math.pi * freq * 2.0 * t) +
                0.20 * math.sin(2.0 * math.pi * freq * 3.0 * t) +
                0.10 * math.sin(2.0 * math.pi * freq * 4.2 * t)
            ) * env * volume
            add_sample(start_idx + i, s * 0.4, s * 0.6)

    def add_celebration_chord(time_sec, volume=0.35):
        notes = [523.25, 659.25, 783.99, 987.77, 1046.50]
        for n_idx, freq in enumerate(notes):
            st = time_sec + n_idx * 0.07
            start_idx = int(st * SAMPLE_RATE)
            num_s = int(2.2 * SAMPLE_RATE)
            for i in range(num_s):
                t = i / SAMPLE_RATE
                env = math.exp(-2.2 * t)
                s = (math.sin(2.0 * math.pi * freq * t) + 0.3 * math.sin(2.0 * math.pi * freq * 2.0 * t)) * env * (volume / len(notes))
                pan = (n_idx / (len(notes) - 1)) * 0.6 + 0.2
                add_sample(start_idx + i, s * (1.0 - pan), s * pan)

    def add_data_stream_hum(start_sec, end_sec):
        start_idx = int(start_sec * SAMPLE_RATE)
        end_idx = int(end_sec * SAMPLE_RATE)
        dur = end_sec - start_sec
        for i in range(end_idx - start_idx):
            t = i / SAMPLE_RATE
            pct = t / dur
            freq = 130.0 + 80.0 * pct
            pulse = 0.5 + 0.5 * math.sin(2.0 * math.pi * 12.0 * t)
            s = math.sin(2.0 * math.pi * freq * t) * pulse * 0.035 + math.sin(2.0 * math.pi * freq * 2.0 * t) * 0.015
            add_sample(start_idx + i, s * 0.55, s * 0.45)

    # Ambient Background (Subtle background atmosphere)
    chords = [
        [146.83, 174.61, 220.00, 261.63],
        [196.00, 246.94, 293.66],
        [130.81, 164.81, 196.00, 246.94],
        [110.00, 130.81, 164.81, 196.00]
    ]
    chord_len = 4.75
    for i in range(total_samples):
        t = i / SAMPLE_RATE
        chord_idx = int((t % (len(chords) * chord_len)) / chord_len)
        chord_t = t % chord_len
        current_chord = chords[chord_idx]
        pad_env = math.sin(math.pi * (chord_t / chord_len)) ** 1.5
        pad = sum(math.sin(2.0 * math.pi * f * t) for f in current_chord) * (0.008 / len(current_chord)) * pad_env
        beat_phase = (t % 0.667) / 0.667
        bass_env = math.exp(-8.0 * beat_phase)
        bass_freq = current_chord[0] * 0.5
        bass = math.sin(2.0 * math.pi * bass_freq * t) * bass_env * 0.020
        add_sample(i, (pad + bass) * 0.5, (pad + bass) * 0.5)

    # Command 1: ventox list
    cmd1 = "ventox list"
    for idx, ch in enumerate(cmd1):
        t = (45 + idx * (40 / len(cmd1))) / FPS
        add_mech_keystroke(t, char=ch)
    add_mech_keystroke(85 / FPS, is_enter=True)
    add_ding(105 / FPS, freq=880.0, volume=0.22)
    add_ding(115 / FPS, freq=1046.5, volume=0.24)

    # Command 2: ventox --dry-run --iso win11.iso --device /dev/sdb
    cmd2 = "ventox --dry-run --iso win11.iso --device /dev/sdb"
    for idx, ch in enumerate(cmd2):
        t = (165 + idx * (45 / len(cmd2))) / FPS
        add_mech_keystroke(t, char=ch)
    add_mech_keystroke(210 / FPS, is_enter=True)
    add_ding(235 / FPS, freq=880.0, volume=0.22)
    add_ding(250 / FPS, freq=1046.5, volume=0.22)
    add_ding(265 / FPS, freq=1318.5, volume=0.28)

    # Command 3: sudo ventox --iso win11.iso --device /dev/sdb --verify
    cmd3 = "sudo ventox --iso win11.iso --device /dev/sdb --verify"
    for idx, ch in enumerate(cmd3):
        t = (315 + idx * (50 / len(cmd3))) / FPS
        add_mech_keystroke(t, char=ch)
    add_mech_keystroke(365 / FPS, is_enter=True)
    add_ding(390 / FPS, freq=880.0, volume=0.22)
    add_ding(405 / FPS, freq=1046.5, volume=0.24)
    for b_idx in range(4):
        add_ding((425 / FPS) + b_idx * 0.08, freq=1200.0 + b_idx * 150.0, volume=0.18, duration=0.2)
    add_data_stream_hum(440 / FPS, 660 / FPS)
    add_mech_keystroke(675 / FPS, is_enter=True)
    add_ding(695 / FPS, freq=1046.5, volume=0.30)
    add_celebration_chord(705 / FPS, volume=0.45)

    # Command 4: sudo ventox format --device /dev/sdb --fs exfat --label "MY_USB"
    cmd4 = 'sudo ventox format --device /dev/sdb --fs exfat --label "MY_USB"'
    for idx, ch in enumerate(cmd4):
        t = (775 + idx * (60 / len(cmd4))) / FPS
        add_mech_keystroke(t, char=ch)
    add_mech_keystroke(835 / FPS, is_enter=True)
    add_ding(885 / FPS, freq=1174.66, volume=0.28)

    # Command 5: ventox version
    cmd5 = "ventox version"
    for idx, ch in enumerate(cmd5):
        t = (955 + idx * (30 / len(cmd5))) / FPS
        add_mech_keystroke(t, char=ch)
    add_mech_keystroke(985 / FPS, is_enter=True)
    add_celebration_chord(995 / FPS, volume=0.40)

    # Write WAV
    with wave.open(TEMP_AUDIO, 'w') as wav:
        wav.setnchannels(2)
        wav.setsampwidth(2)
        wav.setframerate(SAMPLE_RATE)
        out_bytes = bytearray()
        for s_idx in range(total_samples):
            l_val = math.tanh(left[s_idx])
            r_val = math.tanh(right[s_idx])
            out_bytes.extend(struct.pack('<hh', int(l_val * 32767), int(r_val * 32767)))
        wav.writeframes(out_bytes)
    print("✔ WAV audio generated successfully.")

def main():
    print(f"🎬 Starting Ventox All-Features Terminal Video Render ({WIDTH}x{HEIGHT} @ {FPS}fps, {TOTAL_SECONDS:.1f}s)...")
    
    synthesize_audio()

    base_frame = draw_window_frame()

    ffmpeg_cmd = [
        "ffmpeg", "-y",
        "-f", "rawvideo",
        "-vcodec", "rawvideo",
        "-s", f"{WIDTH}x{HEIGHT}",
        "-pix_fmt", "rgb24",
        "-r", str(FPS),
        "-i", "-",
        "-c:v", "libx264",
        "-pix_fmt", "yuv420p",
        "-preset", "faster",
        "-crf", "18",
        TEMP_RAW_VIDEO
    ]
    proc = subprocess.Popen(ffmpeg_cmd, stdin=subprocess.PIPE)

    for f in range(TOTAL_FRAMES):
        im = base_frame.copy()
        draw = ImageDraw.Draw(im)

        lines = get_terminal_content(f)
        if len(lines) > MAX_LINES:
            lines = lines[-MAX_LINES:]

        curr_y = START_Y
        for line_spans in lines:
            curr_x = START_X
            for text, color, is_bold in line_spans:
                font = font_term_bold if is_bold else font_term
                draw.text((curr_x, curr_y), text, font=font, fill=color)
                bbox = draw.textbbox((0, 0), text, font=font)
                curr_x += (bbox[2] - bbox[0])
            curr_y += LINE_HEIGHT

        proc.stdin.write(im.tobytes())

        if f % 100 == 0 or f == TOTAL_FRAMES - 1:
            sys.stdout.write(f"\rRendering Visuals: {f + 1}/{TOTAL_FRAMES} frames ({(f + 1) / TOTAL_FRAMES * 100:.1f}%)")
            sys.stdout.flush()

    proc.stdin.close()
    proc.wait()
    print("\n✔ Visual stream rendered.")

    # Remux video and audio into Mastered Final MP4
    print("Muxing H.264 video and stereo AAC audio...")
    mux_cmd = [
        "ffmpeg", "-y",
        "-i", TEMP_RAW_VIDEO,
        "-i", TEMP_AUDIO,
        "-c:v", "copy",
        "-c:a", "aac",
        "-b:a", "256k",
        "-map", "0:v:0",
        "-map", "1:a:0",
        "-shortest",
        FINAL_MP4
    ]
    subprocess.run(mux_cmd, check=True)
    print("✔ Mastered MP4 generated at:", FINAL_MP4)

    # Generate Animated GIF
    print("Generating animated GIF preview...")
    gif_cmd = [
        "ffmpeg", "-y",
        "-i", FINAL_MP4,
        "-vf", "fps=10,scale=860:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=bayer",
        FINAL_GIF
    ]
    subprocess.run(gif_cmd, check=True)
    print("✔ GIF preview generated at:", FINAL_GIF)

    # Copy to Desktop root for easy access
    shutil.copy2(FINAL_MP4, os.path.join(DEST_DIR_DESKTOP, "ventox-showcase.mp4"))
    shutil.copy2(FINAL_GIF, os.path.join(DEST_DIR_DESKTOP, "ventox-showcase.gif"))
    print(f"✔ Also copied to Desktop root: {os.path.join(DEST_DIR_DESKTOP, 'ventox-showcase.mp4')}")

    # Clean up temp raw video
    if os.path.exists(TEMP_RAW_VIDEO):
        os.remove(TEMP_RAW_VIDEO)

    print("\n🎉 ALL DONE! Video make complete.")

if __name__ == "__main__":
    main()
