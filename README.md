# Capcut-subtitle-json-to-srt

**A utility designed to extract subtitles from CapCut desktop project drafts into a standard `.srt` file.**

This tool interacts directly with the local project files created by the CapCut desktop application to read and process subtitle data.

## Prerequisites

*   CapCut Desktop application installed.
*   Access to the CapCut project data folder on your system.

## Step 1: Locate Your CapCut Project Folder

CapCut saves project drafts in specific directories. You need the path to the *specific project* you want to extract subtitles from.

*   **Default Base Locations:**
    *   **Windows:** `C:\Users\<YourUsername>\AppData\Local\CapCut\User Data\Projects\com.lveditor.draft\`
    *   **macOS:** `/Users/<YourUsername>/Movies/CapCut/User Data/Projects/com.lveditor.draft`
    *(Replace `<YourUsername>` with your actual username)*

*   **Finding Your Project:**
    1.  Navigate to the base location above.
    2.  Inside `com.lveditor.draft`, you'll find subfolders, each usually corresponding to a single CapCut project (often with long, unique names like `12345678-ABCD-1234-ABCD-1234567890AB`).
    3.  Identify the folder for the project you want. Check modification dates or look inside for project details if unsure.
    4.  **Copy the full path** to that specific project folder.

## Step 2: Configure the Tool

1.  In the same directory where `capcut-subtitle.exe` (or the executable) is located, create a plain text file named `file-path.txt`.
2.  Open `file-path.txt` with a text editor (like Notepad, TextEdit).
3.  Paste the **full path** to the CapCut project folder (copied in Step 1) into this file.
    *   *Example (Windows):* `C:\Users\MyUser\AppData\Local\CapCut\User Data\Projects\com.lveditor.draft\12345678-ABCD-1234-ABCD-1234567890AB`
    *   *Example (macOS):* `/Users/MyUser/Movies/CapCut/User Data/Projects/com.lveditor.draft/FEDCBA98-4321-DCBA-4321-BA9876543210FE`
4.  Ensure there is **nothing else** in the file – just the single line containing the path.
5.  Save and close `file-path.txt`.

## Step 3: Run the Tool

1.  Double-click `capcut-subtitle.exe` (or the actual executable file name).
2.  The tool will read the project path from `file-path.txt`, find the project's subtitle data, and extract it.

## Expected Outcome

*   A subtitle file named `subtitles.srt` will be created in the **same directory** as the `capcut-subtitle.exe` executable. This file contains the extracted subtitles in the standard SubRip Text format, ready for use in video players or other editing software.

## Troubleshooting

*   Ensure the path in `file-path.txt` is absolutely correct and points to a valid CapCut project folder containing project data (like `draft_info.json`).
*   Make sure `file-path.txt` is in the *same directory* as the executable.
*   Ensure the CapCut project actually contains subtitles.
*   Consider closing the CapCut application before running the tool to avoid potential file access conflicts.

## How to Build

```
go build -trimpath -ldflags="-s -w" -o capcut-subtitle-json-to-srt.exe .
```
