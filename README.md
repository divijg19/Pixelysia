# `Pixelysia`
> SDDM Theme System

`Pixelysia` is a curated, multi-theme SDDM setup that uses a dispatcher to select between multiple themes at login. It is designed to be portable, reproducible, and easy to install.

---

## Overview

Instead of a single static theme, `Pixelysia` loads one of several themes at runtime using a weighted selection system. This provides variety while maintaining a consistent overall style.

The repository includes:

* Multiple SDDM themes
* A dispatcher (`Main.qml`) that selects a theme
* Bundled fonts required by the themes
* An installation script

---

## Features

* Weighted theme selection via dispatcher
* Multiple theme groups (pixel, nier, sword, enfield, etc.)
* No runtime font loading (all fonts installed system-wide)
* No session selector UI (assumes a default session)
* Consistent username and password input behavior
* Cleaned QML (no unused components or font loaders)

---

## Directory Structure

```text
Pixelysia/
├── Main.qml              # Dispatcher (entry point)
├── metadata.desktop      # SDDM metadata
├── themes/               # All included themes
│   ├── enfield/
│   ├── forest/
│   ├── nier-automata/
│   ├── pixel-*/
│   ├── star-rail/
│   ├── sword/
│   └── tui/
├── fonts/                # Bundled fonts
├── install.sh            # Installation script
├── .gitattributes        # Git LFS configuration
└── README.md
```

---

## Requirements

* SDDM (Qt6 build recommended)
* Linux system using SDDM as display manager
* `git` and `git-lfs` (for cloning the repository)

Some themes require:

```bash
sudo dnf install qt6-qt5compat
```

---

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/divijg19/Pixelysia.git
cd Pixelysia
```

---

### 2. Run install script

```bash
./install.sh
```

This will:

* Install fonts to `~/.local/share/fonts`
* Copy the theme to `/usr/share/sddm/themes/Pixelysia`
* Set `Pixelysia` as the active SDDM theme

---

### 3. Restart SDDM

```bash
sudo systemctl restart sddm
```

---

## Configuration

### Theme selection

Theme selection is controlled by:

```text
Main.qml
```

The dispatcher uses a weighted pool. You can adjust probabilities by modifying the number of entries for each theme.

---

### Fonts

Fonts are bundled under:

```text
fonts/
```

They are installed to:

```text
/usr/share/fonts
```

No QML font loading is used. All themes rely on system fonts.

---

### SDDM configuration

The install script writes:

```text
/etc/sddm.conf.d/theme.conf
```

With:

```ini
[Theme]
Current=Pixelysia
```

---

## Development

### Editing themes

Edit files under:

```text
themes/
```

After changes:

```bash
sudo cp -r Pixelysia /usr/share/sddm/themes/Pixelysia
```

Then test:

```bash
sddm-greeter-qt6 --test-mode --theme /usr/share/sddm/themes/Pixelysia
```

---

### Dispatcher

The dispatcher (`Main.qml`) selects which theme to load. It supports:

* Grouped themes (pixel, tui, etc.)
* Weighted randomness

---

## Git LFS

Large assets (videos, images) are tracked using Git LFS.

Setup:

```bash
git lfs install
git lfs track "*.mp4"
git lfs track "*.png"
```

---

## Notes

* The session selector UI has been removed; a default session must be configured in SDDM.
* Themes are independent; shared behavior was applied via batch refactoring rather than runtime centralization.
* Fonts are required for correct rendering.

---

## Troubleshooting

### Theme not applied

Check:

```bash
cat /etc/sddm.conf.d/theme.conf
```

---

### Test manually

```bash
sddm-greeter-qt6 --test-mode --theme /usr/share/sddm/themes/Pixelysia
```

---

### Fonts not applied

```bash
fc-cache -fv
fc-list | grep -i pixelify
```
