# DraftIt

DraftIt is a simple pixel-based sketch pad built with [Ebiten](https://ebiten.org/) that offers quick tools for jotting down ideas. It provides a persistent toolbar for switching tools, adjustable brush sizes, and a built-in save dialog that crops output to just the area you draw on.

## Features
- Brush, pixel eraser, and stroke eraser tools with adjustable sizes via sliders.
- Text tool for placing, editing, dragging, resizing, and deleting labeled boxes on the canvas.
- Save dialog with filename editing and directory navigation that writes a cropped `.png` of only the drawn content.
- Undo/redo support for strokes and text placement with `Ctrl+Z` / `Ctrl+R` (or `Cmd` on macOS).
- Canvas panning with the right mouse button plus vertical scrolling via the mouse wheel or arrow keys.
- Clear confirmation dialog to reset the canvas without closing the app.

## Controls
- **Mouse**
  - Left click/drag to draw with the current brush or eraser.
  - Left click to place or select text; drag to move selected text.
  - Right click/drag to pan the view; mouse wheel or arrow keys scroll vertically.
  - Use the top toolbar buttons to switch modes (Brush, Pixel Eraser, Stroke Eraser, Text), save the drawing, or clear the canvas.
  - Sliders adjust brush, eraser, and text sizes.
- **Keyboard**
  - `Ctrl+Z` / `Cmd+Z` to undo; `Ctrl+R` / `Cmd+R` to redo.
  - `Enter` finishes editing a text box; `Delete` removes the selected text box; `Backspace` deletes text while editing.
  - `Esc` closes the save dialog.

## Running the app
1. Install [Go 1.22+](https://go.dev/dl/).
2. From the project root, run:
   ```sh
   go run .
   ```
   This starts a windowed sketch pad with the toolbar at the top.

## Building binaries
The included `Makefile` builds platform-specific binaries and embeds the Windows icon when available.
- Build Linux amd64 and Windows amd64 binaries:
  ```sh
  make build
  ```
- Clean build artifacts:
  ```sh
  make clean
  ```

Cross-compiling for Windows requires `CGO_ENABLED=1` (the default here) and the `.ico` asset in `assets/`.
